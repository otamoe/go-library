package libraft

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/dgraph-io/badger/v3"
	goLog "github.com/ipfs/go-log/v2"
	draftio "github.com/lni/dragonboat/v3/raftio"
	draftpb "github.com/lni/dragonboat/v3/raftpb"
	"go.uber.org/zap"
)

const RaftKey = byte('r')

// 启动信息 节点前缀
var (
	entryKeyPrefix = []byte{RaftKey, 1}

	persistentStateKeyPrefix = []byte{RaftKey, 2}

	maxIndexKeyPrefix = []byte{RaftKey, 3}

	nodeInfoKeyPrefix = []byte{RaftKey, 4}

	snapshotKeyPrefix = []byte{RaftKey, 5}

	bootstrapKeyPrefix = []byte{RaftKey, 6}

	entryBatchKeyPrefix = []byte{RaftKey, 7}

	stateMachineKeyPrefix = []byte{RaftKey, 128}
)

type (
	LogDB struct {
		db *badger.DB

		logger *zap.Logger

		cache *cache

		mu sync.Mutex
	}
)

func NewLogDB(db *badger.DB) (logDB *LogDB, err error) {
	logDB = &LogDB{
		db:     db,
		logger: goLog.Logger("raft.logdb").Desugar(),
		cache:  newCache(),
	}
	return
}

// logdb 名
func (logDB *LogDB) Name() string {
	return "badger"
}

// 关闭 logdb
func (logDB *LogDB) Close() {

}

// 数据库二进制 版本
func (logDB *LogDB) BinaryFormat() uint32 {
	return draftio.LogDBBinVersion
}

// 列出 启动 节点
func (logDB *LogDB) ListNodeInfo() (nodeInfos []draftio.NodeInfo, err error) {
	err = logDB.db.View(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.IteratorOptions{Prefix: bootstrapKeyPrefix})
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			clusterID, nodeID, errp := parseBootstrapKey(key)
			if errp != nil {
				logDB.logger.Error("ListNodeInfo", zap.Error(err), zap.Binary("key", key))
			} else {
				nodeInfos = append(nodeInfos, draftio.GetNodeInfo(clusterID, nodeID))
			}
		}
		return
	})
	return
}

// 保存 启动 信息
func (logDB *LogDB) SaveBootstrapInfo(clusterID uint64, nodeID uint64, bootstrap draftpb.Bootstrap) (err error) {
	var data []byte
	if data, err = bootstrap.Marshal(); err != nil {
		logDB.logger.Panic("SaveBootstrapInfo", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID))
	}
	err = logDB.db.Update(func(txn *badger.Txn) (err error) {
		err = dbSet(txn, newBootstrapKey(clusterID, nodeID), data)
		return
	})

	return
}

// 读取 启动节点
func (logDB *LogDB) GetBootstrapInfo(clusterID uint64, nodeID uint64) (bootstrap draftpb.Bootstrap, err error) {
	var value []byte
	err = logDB.db.View(func(txn *badger.Txn) (err error) {
		var item *badger.Item
		if item, err = txn.Get(newBootstrapKey(clusterID, nodeID)); err != nil {
			return
		}
		if value, err = item.ValueCopy(nil); err != nil {
			logDB.logger.Panic("GetBootstrapInfo", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID))
		}
		return
	})

	if err == badger.ErrKeyNotFound || len(value) == 0 {
		err = draftio.ErrNoBootstrapInfo
		return
	}
	if err = bootstrap.Unmarshal(value); err != nil {
		logDB.logger.Panic("GetBootstrapInfo", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID))
		return
	}
	return
}

// 保存 raft 状态
func (logDB *LogDB) SaveRaftState(updates []draftpb.Update, shardID uint64) (err error) {
	if len(updates) == 0 {
		return
	}
	err = logDB.db.Update(func(txn *badger.Txn) (err error) {
		for _, update := range updates {

			// 保存 状态
			if err = logDB.saveState(txn, update); err != nil {
				return
			}

			// 不是空 快照
			if !draftpb.IsEmptySnapshot(update.Snapshot) {
				if len(update.EntriesToSave) > 0 {
					// raft/inMemory makes sure such entries no longer need to be saved
					lastIndex := update.EntriesToSave[len(update.EntriesToSave)-1].Index
					if update.Snapshot.Index > lastIndex {
						logDB.logger.Panic("max index not handled, %d, %d", zap.Uint64("Index", update.Snapshot.Index), zap.Uint64("lastIndex", lastIndex), zap.Uint64("clusterID", update.ClusterID), zap.Uint64("nodeID", update.NodeID))
					}
				}

				// 保存快照
				if err = logDB.saveSnapshot(txn, update); err != nil {
					return
				}
				// 保存最大 index
				if err = logDB.setMaxIndex(txn, update, update.Snapshot.Index); err != nil {
					return
				}
			}
		}
		if err = logDB.saveEntries(txn, updates); err != nil {
			return
		}
		return
	})
	return
}

// entries 代送
func (logDB *LogDB) IterateEntries(entrys []draftpb.Entry, size uint64, clusterID uint64, nodeID uint64, lowIndex uint64, highIndex uint64, maxSize uint64) (rentrys []draftpb.Entry, rsize uint64, err error) {
	err = logDB.db.View(func(txn *badger.Txn) (err error) {
		var maxIndex uint64
		if maxIndex, err = logDB.getMaxIndex(txn, clusterID, nodeID); err != nil {
			if err == draftio.ErrNoSavedLog {
				rentrys = entrys
				rsize = size
				return
			}
			logDB.logger.Panic("IterateEntries", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID))
		}

		if highIndex > maxIndex+1 {
			highIndex = maxIndex + 1
		}

		it := txn.NewIterator(badger.IteratorOptions{Prefix: newKey(entryKeyPrefix, clusterID, nodeID)})
		defer it.Close()

		// 遍历
		expectedIndex := lowIndex
		for it.Seek(newEntryKey(clusterID, nodeID, lowIndex)); maxSize >= size && it.Valid(); it.Next() {

			item := it.Item()

			var data []byte
			if data, err = item.ValueCopy(nil); err != nil {
				logDB.logger.Panic("IterateEntries", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID), zap.Binary("key", item.KeyCopy(nil)))
			}
			var entry draftpb.Entry
			if err = entry.Unmarshal(data); err != nil {
				logDB.logger.Panic("IterateEntries", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID), zap.Binary("key", item.KeyCopy(nil)))
			}

			// 大于等于 highIndex
			if entry.Index >= highIndex {
				break
			}

			// index 不是自增
			if entry.Index != expectedIndex {
				break
			}

			size += uint64(entry.SizeUpperLimit())
			entrys = append(entrys, entry)
			expectedIndex++
		}
		return
	})

	if err == draftio.ErrNoSavedLog {
		rentrys = entrys
		rsize = size
		return
	}

	if err != nil {
		return
	}

	rentrys = entrys
	rsize = size

	return
}

// ReadRaftState返回在日志数据库中找到的持久化raft状态。
func (logDB *LogDB) ReadRaftState(clusterID uint64, nodeID uint64, snapshotIndex uint64) (raftState draftio.RaftState, err error) {
	err = logDB.db.View(func(txn *badger.Txn) (err error) {
		var firstIndex uint64
		var length uint64
		if firstIndex, length, err = logDB.getRange(txn, clusterID, nodeID, snapshotIndex); err != nil {
			return
		}

		var state draftpb.State
		if state, err = logDB.getState(txn, clusterID, nodeID); err != nil {
			return
		}

		raftState = draftio.RaftState{
			State:      state,
			FirstIndex: firstIndex,
			EntryCount: length,
		}
		return
	})
	if err != nil {
		return
	}
	return
}

// RemoveEntriesTo 删除与指定Raft节点关联的条目
//到指定的索引。
func (logDB *LogDB) RemoveEntriesTo(clusterID uint64, nodeID uint64, index uint64) (err error) {
	err = logDB.db.Update(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.IteratorOptions{Prefix: newKey(entryKeyPrefix, clusterID, nodeID)})
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {

			// 读取 key
			key := it.Item().KeyCopy(nil)

			// 读取 key 的 index
			var currentIndex uint64
			if _, _, currentIndex, err = parseEntryKey(key); err != nil {
				logDB.logger.Panic("RemoveEntriesTo", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID), zap.Binary("key", key))
			}

			// 大于 等于 break
			if currentIndex >= index {
				break
			}

			// 删除
			err = dbDelete(txn, key)
		}
		return
	})
	return
}

// CompactEntriesTo 回收用于存储的底层存储空间
// 达到指定索引的条目。
func (logDB *LogDB) CompactEntriesTo(clusterID uint64, nodeID uint64, index uint64) (<-chan struct{}, error) {
	done := make(chan struct{})
	close(done)
	return done, nil
}

// 保存快照
func (logDB *LogDB) SaveSnapshots(updates []draftpb.Update) (err error) {
	if len(updates) == 0 {
		return
	}
	err = logDB.db.Update(func(txn *badger.Txn) (err error) {
		for _, update := range updates {
			if err = logDB.saveSnapshot(txn, update); err != nil {
				return
			}
		}
		return
	})
	return
}

// 删除快照
func (logDB *LogDB) DeleteSnapshot(clusterID uint64, nodeID uint64, index uint64) (err error) {
	err = logDB.db.Update(func(txn *badger.Txn) (err error) {
		if err = dbDelete(txn, newSnapshotKey(clusterID, nodeID, index)); err != nil {
			if err != badger.ErrKeyNotFound {
				return
			}
			err = nil
		}
		return
	})
	return
}

// ListSnapshots 列出与指定的关联的可用快照
// 索引范围 (0, index].
func (logDB *LogDB) ListSnapshots(clusterID uint64, nodeID uint64, index uint64) (snapshots []draftpb.Snapshot, err error) {
	err = logDB.db.View(func(txn *badger.Txn) (err error) {
		it := txn.NewIterator(badger.IteratorOptions{Prefix: newKey(snapshotKeyPrefix, clusterID, nodeID)})
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.KeyCopy(nil)
			var data []byte
			if data, err = item.ValueCopy(nil); err != nil {
				logDB.logger.Panic("ListSnapshots", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID), zap.Binary("key", key))
			}

			snapshot := draftpb.Snapshot{}
			if err = snapshot.Unmarshal(data); err != nil {
				logDB.logger.Panic("ListSnapshots", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID), zap.Binary("key", key))
			}
			// 快照大于等于
			if snapshot.Index >= index {
				break
			}
			snapshots = append(snapshots, snapshot)
		}
		return
	})
	return
}

// RemoveNodeData 删除与指定节点关联的所有数据。
func (logDB *LogDB) RemoveNodeData(clusterID uint64, nodeID uint64) (err error) {
	var snapshots []draftpb.Snapshot
	if snapshots, err = logDB.ListSnapshots(clusterID, nodeID, math.MaxUint64); err != nil {
		return
	}
	err = logDB.db.Update(func(txn *badger.Txn) (err error) {
		if err = logDB.saveRemoveNodeData(txn, snapshots, clusterID, nodeID); err != nil {
			return
		}
		return
	})
	if err == nil {
		logDB.cache.setMaxIndex(clusterID, nodeID, 0)
	}
	if err = logDB.RemoveEntriesTo(clusterID, nodeID, math.MaxUint64); err != nil {
		return
	}

	return
}

// ImportSnapshot 通过创建所有需要的来导入指定的快照
// logdb 中的元数据。
func (logDB *LogDB) ImportSnapshot(snapshot draftpb.Snapshot, nodeID uint64) (err error) {
	if snapshot.Type == draftpb.UnknownStateMachine {
		logDB.logger.Panic("Unknown state machine type")
	}
	var snapshots []draftpb.Snapshot
	if snapshots, err = logDB.ListSnapshots(snapshot.ClusterId, nodeID, math.MaxUint64); err != nil {
		return
	}

	selectedss := make([]draftpb.Snapshot, 0)
	for _, curss := range snapshots {
		if curss.Index >= snapshot.Index {
			selectedss = append(selectedss, curss)
		}
	}

	bsrec := draftpb.Bootstrap{
		Join: true,
		Type: snapshot.Type,
	}
	state := draftpb.State{
		Term:   snapshot.Term,
		Commit: snapshot.Index,
	}
	err = logDB.db.Update(func(txn *badger.Txn) (err error) {
		if err = logDB.saveRemoveNodeData(txn, selectedss, snapshot.ClusterId, nodeID); err != nil {
			return
		}
		if err = logDB.saveBootstrap(txn, snapshot.ClusterId, nodeID, bsrec); err != nil {
			return
		}
		if err = logDB.saveStateAllocs(txn, snapshot.ClusterId, nodeID, state); err != nil {
			return
		}
		if err = logDB.saveSnapshot(txn, draftpb.Update{ClusterID: snapshot.ClusterId, NodeID: nodeID, Snapshot: snapshot}); err != nil {
			return
		}
		if err = logDB.saveMaxIndex(txn, snapshot.ClusterId, nodeID, snapshot.Index); err != nil {
			return
		}
		return
	})

	return
}

func (logDB *LogDB) saveBootstrap(txn *badger.Txn, clusterID uint64, nodeID uint64, bootstrap draftpb.Bootstrap) (err error) {
	var data []byte
	if data, err = bootstrap.Marshal(); err != nil {
		logDB.logger.Panic("saveBootstrap", zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID))
	}

	if err = dbSet(txn, newBootstrapKey(clusterID, nodeID), data); err != nil {
		return
	}
	return
}

func (logDB *LogDB) getRange(txn *badger.Txn, clusterID uint64, nodeID uint64, snapshotIndex uint64) (firstIndex uint64, length uint64, err error) {
	maxIndex, err := logDB.getMaxIndex(txn, clusterID, nodeID)
	if err == draftio.ErrNoSavedLog {
		return snapshotIndex, 0, nil
	}

	if err != nil {
		return 0, 0, err
	}

	if snapshotIndex == maxIndex {
		return snapshotIndex, 0, nil
	}

	it := txn.NewIterator(badger.IteratorOptions{Prefix: newKey(entryKeyPrefix, clusterID, nodeID)})
	defer it.Close()

	for it.Seek(newEntryKey(clusterID, nodeID, snapshotIndex)); it.Valid(); it.Next() {
		if firstIndex == 0 {
			var entry draftpb.Entry
			var data []byte
			if data, err = it.Item().ValueCopy(nil); err != nil {
				logDB.logger.Panic("getRange", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID), zap.Uint64("snapshotIndex", snapshotIndex))
			}
			if err = entry.Unmarshal(data); err != nil {
				logDB.logger.Panic("getRange", zap.Error(err), zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID), zap.Uint64("snapshotIndex", snapshotIndex))
			}
			firstIndex = entry.Index
			break
		}
	}

	if firstIndex == 0 && maxIndex != 0 {
		logDB.logger.Panic(
			"getRange",
			zap.Uint64("clusterID", clusterID),
			zap.Uint64("nodeID", nodeID),
			zap.Uint64("snapshotIndex", snapshotIndex),
			zap.Uint64("firstIndex", firstIndex),
			zap.Uint64("maxIndex", maxIndex),
		)
	}
	if firstIndex > 0 {
		length = maxIndex - firstIndex + 1
	}
	return
}

func (logDB *LogDB) saveStateAllocs(txn *badger.Txn, clusterID uint64, nodeID uint64, state draftpb.State) (err error) {
	var data []byte
	if data, err = state.Marshal(); err != nil {
		logDB.logger.Panic("saveStateAllocs", zap.Uint64("clusterID", clusterID), zap.Uint64("nodeID", nodeID))
	}

	if err = dbSet(txn, newPersistentStateKey(clusterID, nodeID), data); err != nil {
		return
	}
	return
}

func (logDB *LogDB) getState(txn *badger.Txn, clusterID uint64, nodeID uint64) (state draftpb.State, err error) {
	var item *badger.Item
	if item, err = txn.Get(newPersistentStateKey(clusterID, nodeID)); err != nil {
		if err == badger.ErrKeyNotFound {
			err = draftio.ErrNoSavedLog
		}
		return
	}

	var data []byte
	if data, err = item.ValueCopy(nil); err != nil {
		return
	}
	if err = state.Unmarshal(data); err != nil {
		return
	}
	return
}

func (logDB *LogDB) saveState(txn *badger.Txn, update draftpb.Update) (err error) {
	// 空状态
	if draftpb.IsEmptyState(update.State) {
		return
	}

	// 写入缓存状态
	if !logDB.cache.setState(update.ClusterID, update.NodeID, update.State) {
		return
	}
	var data []byte
	if data, err = update.State.Marshal(); err != nil {
		logDB.logger.Panic("saveState", zap.Error(err))
	}

	// 状态
	if err = dbSet(txn, newPersistentStateKey(update.ClusterID, update.NodeID), data); err != nil {
		logDB.logger.Panic("saveState", zap.Error(err))
	}

	return
}

func (logDB *LogDB) saveSnapshot(txn *badger.Txn, update draftpb.Update) (err error) {
	if draftpb.IsEmptySnapshot(update.Snapshot) {
		return
	}

	// 保存 快照
	var data []byte
	if data, err = update.Snapshot.Marshal(); err != nil {
		logDB.logger.Panic("saveSnapshot", zap.Error(err))
	}
	if err = dbSet(txn, newSnapshotKey(update.ClusterID, update.NodeID, update.Snapshot.Index), data); err != nil {
		logDB.logger.Panic("saveSnapshot", zap.Error(err))
	}

	return
}

func (logDB *LogDB) setMaxIndex(txn *badger.Txn, ud draftpb.Update, maxIndex uint64) (err error) {
	logDB.cache.setMaxIndex(ud.ClusterID, ud.NodeID, maxIndex)
	err = logDB.saveMaxIndex(txn, ud.ClusterID, ud.NodeID, maxIndex)
	return
}

func (logDB *LogDB) getMaxIndex(txn *badger.Txn, clusterID uint64, nodeID uint64) (maxIndex uint64, err error) {
	if v, ok := logDB.cache.getMaxIndex(clusterID, nodeID); ok {
		maxIndex = v
		return
	}

	var item *badger.Item
	if item, err = txn.Get(newMaxIndexKey(clusterID, nodeID)); err != nil {
		if err == badger.ErrKeyNotFound {
			err = draftio.ErrNoSavedLog
		}
		return
	}
	var data []byte
	if data, err = item.ValueCopy(nil); err != nil {
		return
	}
	if len(data) == 0 {
		err = draftio.ErrNoSavedLog
		return
	}

	maxIndex = binary.BigEndian.Uint64(data)

	return
}

func (logDB *LogDB) saveMaxIndex(txn *badger.Txn, clusterID uint64, nodeID uint64, index uint64) (err error) {
	// 保存 快照 index
	var data []byte
	data = make([]byte, 8)
	binary.BigEndian.PutUint64(data, index)
	if err = dbSet(txn, newMaxIndexKey(clusterID, nodeID), data); err != nil {
		logDB.logger.Panic("saveMaxIndex", zap.Error(err))
	}

	return
}

func (logDB *LogDB) getEntry(txn *badger.Txn, clusterID uint64, nodeID uint64, index uint64) (entry draftpb.Entry, err error) {
	var item *badger.Item
	if item, err = txn.Get(newEntryKey(clusterID, nodeID, index)); err != nil {
		if err == badger.ErrKeyNotFound {
			err = nil
		}
		return
	}
	var data []byte
	if data, err = item.ValueCopy(nil); err != nil {
		return
	}

	if err = entry.Unmarshal(data); err != nil {
		return
	}

	return
}

func (logDB *LogDB) saveEntries(txn *badger.Txn, updates []draftpb.Update) (err error) {
	for _, update := range updates {
		if len(update.EntriesToSave) > 0 {
			idx := 0
			var maxIndex uint64
			for idx < len(update.EntriesToSave) {
				ent := update.EntriesToSave[idx]

				var data []byte
				if data, err = ent.Marshal(); err != nil {
					logDB.logger.Panic("SaveRaftState", zap.Error(err))
				}

				dbSet(txn, newEntryKey(update.ClusterID, update.NodeID, ent.Index), data)
				if ent.Index > maxIndex {
					maxIndex = ent.Index
				}
				idx++
			}

			if maxIndex > 0 {
				if err = logDB.setMaxIndex(txn, update, maxIndex); err != nil {
					return
				}
			}
		}
	}
	return
}

func (logDB *LogDB) saveRemoveNodeData(txn *badger.Txn, snapshots []draftpb.Snapshot, clusterID uint64, nodeID uint64) (err error) {

	// 删除 state
	if err = dbDelete(txn, newPersistentStateKey(clusterID, nodeID)); err != nil {
		if err != badger.ErrKeyNotFound {
			return
		}
		err = nil
	}

	// 删除启动信息
	if err = dbDelete(txn, newBootstrapKey(clusterID, nodeID)); err != nil {
		if err != badger.ErrKeyNotFound {
			return
		}
		err = nil
	}
	// 删除 max index
	if err = dbDelete(txn, newMaxIndexKey(clusterID, nodeID)); err != nil {
		if err != badger.ErrKeyNotFound {
			return
		}
		err = nil
	}

	// 删除快照
	for _, snapshot := range snapshots {
		if err = dbDelete(txn, newSnapshotKey(clusterID, nodeID, snapshot.Index)); err != nil {
			return
		}
	}
	return
}

func newKey(prefix []byte, clusterID uint64, nodeID uint64) (key []byte) {
	l := len(prefix)
	key = make([]byte, l+16)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[l:], clusterID)
	binary.BigEndian.PutUint64(key[l+8:], nodeID)
	return
}

func parseKey(prefix []byte, key []byte) (clusterID uint64, nodeID uint64, err error) {
	l := len(prefix)
	// 长度不匹配
	if len(key) != (l + 16) {
		err = errors.New("invalid key")
		return
	}
	// 前缀不匹配
	if !bytes.Equal(prefix, key[:l]) {
		err = errors.New("invalid key")
		return
	}
	clusterID = binary.BigEndian.Uint64(key[l:])
	nodeID = binary.BigEndian.Uint64(key[l+8:])
	return
}

func newBootstrapKey(clusterID uint64, nodeID uint64) (key []byte) {
	return newKey(bootstrapKeyPrefix, clusterID, nodeID)
}
func parseBootstrapKey(key []byte) (clusterID uint64, nodeID uint64, err error) {
	return parseKey(bootstrapKeyPrefix, key)
}
func newPersistentStateKey(clusterID uint64, nodeID uint64) (key []byte) {
	return newKey(persistentStateKeyPrefix, clusterID, nodeID)
}
func parsePersistentStateKey(key []byte) (clusterID uint64, nodeID uint64, err error) {
	return parseKey(persistentStateKeyPrefix, key)
}
func newMaxIndexKey(clusterID uint64, nodeID uint64) (key []byte) {
	return newKey(maxIndexKeyPrefix, clusterID, nodeID)
}
func parseMaxIndexKey(key []byte) (clusterID uint64, nodeID uint64, err error) {
	return parseKey(maxIndexKeyPrefix, key)
}

func newSnapshotKey(clusterID uint64, nodeID uint64, index uint64) (key []byte) {
	prefix := newKey(snapshotKeyPrefix, clusterID, nodeID)
	l := len(prefix)
	key = make([]byte, l+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[l:], index)
	return
}
func parseSnapshotKey(key []byte) (clusterID uint64, nodeID uint64, index uint64, err error) {
	l := len(snapshotKeyPrefix)
	if len(key) != (l + 24) {
		err = errors.New("invalid key")
		return
	}
	if clusterID, nodeID, err = parseKey(snapshotKeyPrefix, key[0:l+16]); err != nil {
		return
	}
	index = binary.BigEndian.Uint64(key[l+16:])
	return
}

func newEntryKey(clusterID uint64, nodeID uint64, index uint64) (key []byte) {
	prefix := newKey(entryKeyPrefix, clusterID, nodeID)
	l := len(prefix)
	key = make([]byte, l+8)
	copy(key, prefix)
	binary.BigEndian.PutUint64(key[l:], index)
	return
}
func parseEntryKey(key []byte) (clusterID uint64, nodeID uint64, index uint64, err error) {
	l := len(entryKeyPrefix)
	if len(key) != (l + 24) {
		err = errors.New("invalid key")
		return
	}
	if clusterID, nodeID, err = parseKey(entryKeyPrefix, key[0:l+16]); err != nil {
		return
	}
	index = binary.BigEndian.Uint64(key[l+16:])
	return
}

func dbDelete(txn *badger.Txn, key []byte) (err error) {
	err = txn.Delete(key)
	if err == badger.ErrTxnTooBig {
		if err = txn.Commit(); err != nil {
			return
		}
		err = txn.Delete(key)
	}
	return
}
func dbSet(txn *badger.Txn, key []byte, value []byte) (err error) {
	err = txn.Set(key, value)
	if err == badger.ErrTxnTooBig {
		if err = txn.Commit(); err != nil {
			return
		}
		err = txn.Set(key, value)
	}
	return
}
