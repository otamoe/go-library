package libraft

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"sync"

	"github.com/dgraph-io/badger/v3"
	dstatemachine "github.com/lni/dragonboat/v3/statemachine"
	libraftpb "github.com/otamoe/go-library/raft/pb"
)

type (
	StateMachine struct {
		mux sync.Mutex

		db        *badger.DB
		clusterID uint64
		nodeID    uint64

		snapshots []*badger.Txn

		// 事件
		event *StateMachineEvent

		closed bool
	}
)

const (
	StateMachineResultCodeSuccess uint64 = 0
	StateMachineResultCodeFailure uint64 = 1
)

func NewStateMachine(db *badger.DB, event StateMachineEventFunc) dstatemachine.CreateConcurrentStateMachineFunc {
	return func(clusterID uint64, nodeID uint64) dstatemachine.IConcurrentStateMachine {
		stateMachine := &StateMachine{
			db:        db,
			clusterID: clusterID,
			nodeID:    nodeID,
		}
		stateMachine.event = NewStateMachineEvent(clusterID, nodeID, event)
		go stateMachine.event.Start()

		return stateMachine
	}
}

// 关闭
func (stateMachine *StateMachine) Close() (err error) {
	stateMachine.mux.Lock()
	defer stateMachine.mux.Unlock()

	if stateMachine.closed {
		return
	}
	stateMachine.closed = true

	err = stateMachine.event.Close()
	for _, snapshot := range stateMachine.snapshots {
		snapshot.Discard()
	}
	return
}

// 更新
func (stateMachine *StateMachine) Update(sEntries []dstatemachine.Entry) (rentries []dstatemachine.Entry, err error) {
	for sIndex, sEntry := range sEntries {
		updateRequest := &libraftpb.UpdateRequest{}
		response := &libraftpb.Response{
			Index: -2,
		}

		// 解码
		if err = updateRequest.Unmarshal(sEntry.Cmd); err != nil {
			response.Index = -1
			response.Error = err.Error()
			sEntries[sIndex].Result.Data, _ = response.Marshal()
			sEntries[sIndex].Result.Value = StateMachineResultCodeFailure
			continue
		}

		eventBatch := stateMachine.event.Batch()

		// 遍历 更新
		err = stateMachine.db.Update(func(txn *badger.Txn) (err error) {
			response.Data = make([]*libraftpb.ResponseData, len(updateRequest.Entrys))

			for index, entry := range updateRequest.Entrys {
				if response.Data[index], err = stateMachine.updateEntry(txn, entry, eventBatch, updateRequest.Response); err != nil {
					response.Index = int32(index)
					return
				}
			}

			// 编码 response 错误
			if sEntries[sIndex].Result.Data, err = response.Marshal(); err != nil {
				return
			}
			return
		})

		// txn 有错误
		if err != nil {
			if response.Index == -2 {
				response.Index = -1
			}
			response.Data = nil
			response.Error = err.Error()
			sEntries[sIndex].Result.Data, _ = response.Marshal()
			sEntries[sIndex].Result.Value = StateMachineResultCodeFailure
			continue
		}

		// 成功
		sEntries[sIndex].Result.Value = StateMachineResultCodeSuccess

		// 提交事件
		eventBatch.Commit()
	}
	rentries = sEntries
	return
}

func (stateMachine *StateMachine) Lookup(e interface{}) (out interface{}, err error) {
	response := &libraftpb.Response{
		Index: -2,
	}
	func() {
		out = response
	}()

	lookup, ok := e.(*libraftpb.LookupRequest)
	if !ok {
		err = fmt.Errorf("Invalid query %#v", e)
		response.Error = err.Error()
		response.Index = -1
		return
	}

	// 遍历查询
	response.Data = make([]*libraftpb.ResponseData, len(lookup.Entrys))
	err = stateMachine.db.View(func(txn *badger.Txn) (err error) {
		for index, entry := range lookup.Entrys {
			if response.Data[index], err = stateMachine.lookupEntry(txn, entry, lookup.Response); err != nil {
				response.Index = int32(index)
				return
			}
		}
		return
	})

	if err != nil {
		if response.Index == -2 {
			response.Index = -1
		}
		response.Data = nil
		response.Error = err.Error()
	}
	out = response

	return
}

// 预处理快照
func (stateMachine *StateMachine) PrepareSnapshot() (itxn interface{}, err error) {
	stateMachine.mux.Lock()
	defer stateMachine.mux.Unlock()
	if stateMachine.closed {
		err = errors.New("save snapshot called after Close")
		return
	}
	txn := stateMachine.db.NewTransaction(false)
	stateMachine.snapshots = append(stateMachine.snapshots, txn)
	itxn = txn
	return
}

// 保存快照到其他服务器
func (stateMachine *StateMachine) SaveSnapshot(itxn interface{}, w io.Writer, sfc dstatemachine.ISnapshotFileCollection, stopc <-chan struct{}) (err error) {
	txn, ok := itxn.(*badger.Txn)
	if !ok {
		err = errors.New("txn is incorrect")
		return
	}

	stateMachine.mux.Lock()
	if stateMachine.closed {
		err = errors.New("save snapshot called after Close")
		stateMachine.mux.Unlock()
		return
	}
	stateMachine.mux.Unlock()

	// 预处理快照
	defer func() {
		stateMachine.mux.Lock()
		defer stateMachine.mux.Unlock()
		var snapshots []*badger.Txn
		for _, snapshot := range stateMachine.snapshots {
			if snapshot == txn {
				continue
			}
			snapshots = append(snapshots, txn)
		}
		stateMachine.snapshots = snapshots
	}()

	defer txn.Discard()

	it := txn.NewIterator(badger.IteratorOptions{

		// 预处理 大小 200
		PrefetchSize: 200,

		// 预处理
		PrefetchValues: true,

		// 前缀
		Prefix: stateMachine.NewKey(nil),
	})
	defer it.Close()

	// 写入缓冲区 2m
	bufiow := bufio.NewWriterSize(w, 1024*1024*2)

	// 遍历
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()

		// key 解析
		var key []byte
		if key, err = stateMachine.ParseKey(item.KeyCopy(nil)); err != nil {
			return
		}

		// value 读取
		var value []byte
		if value, err = item.ValueCopy(nil); err != nil {
			return
		}

		pbItem := &libraftpb.Item{
			Key:    key,
			Value:  value,
			Expire: item.ExpiresAt(),
		}
		var dataBody []byte
		if dataBody, err = pbItem.Marshal(); err != nil {
			return
		}

		// data 大小
		dataSize := make([]byte, 4)
		binary.BigEndian.PutUint32(dataSize, uint32(len(dataBody))+4)

		// data hash
		dataHash := make([]byte, 4)
		binary.BigEndian.PutUint32(dataHash, crc32.ChecksumIEEE(dataBody))

		// data 长度
		data := bytes.Join([][]byte{dataSize, dataHash, dataBody}, []byte{})

		select {
		case <-stopc:
			// 停止
			return
		default:
			// 写入
			var n int
			if n, err = bufiow.Write(data); err != nil {
				return
			}
			if n != len(data) {
				err = errors.New("Write size does not match")
				return
			}
		}
	}

	// 刷入
	err = bufiow.Flush()

	return
}

// 恢复快照 (从其他服务器恢复)
func (stateMachine *StateMachine) RecoverFromSnapshot(r io.Reader, sfc []dstatemachine.SnapshotFile, stopc <-chan struct{}) (err error) {
	// 设置读取缓冲区
	bufior := bufio.NewReaderSize(r, 1024*1024*2)

	// 更新 txn
	err = stateMachine.db.Update(func(txn *badger.Txn) (err error) {

		// 删除 当前集群的 item
		if err = stateMachine.recoverDelete(txn); err != nil {
			return
		}

		// 写入 item
		var item *libraftpb.Item
		var size uint32
		for {
			select {
			case <-stopc:
				// 停止
				return
			default:
				// size
				if size, err = stateMachine.recoverDataSize(bufior); err != nil {
					return
				}
			}

			// 读取结束
			if size == 0 {
				return
			}

			select {
			case <-stopc:
				// 停止
				return
			default:
				// 读取 data
				if item, err = stateMachine.recoverItem(bufior, size); err == nil {
					return
				}

				// 设置
				itemEntry := &badger.Entry{
					Key:       stateMachine.NewKey(item.Key),
					Value:     item.Value,
					ExpiresAt: item.Expire,
				}
				if item.Expire != math.MaxUint64 {
					itemEntry.ExpiresAt = item.Expire
				}

				// 提交
				err = txn.SetEntry(itemEntry)
				if err == badger.ErrTxnTooBig {
					if err = txn.Commit(); err == nil {
						err = txn.SetEntry(itemEntry)
					}
				}
				if err != nil {
					return
				}
			}
		}
	})

	if err != nil {
		return
	}

	return
}

// 删除 当前集群的 item
func (stateMachine *StateMachine) recoverDelete(txn *badger.Txn) (err error) {
	it := txn.NewIterator(badger.IteratorOptions{
		// 前缀
		Prefix: stateMachine.NewKey(nil),
	})
	defer it.Close()

	// 遍历
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()

		key := item.KeyCopy(nil)

		err = txn.Delete(key)
		if err == badger.ErrTxnTooBig {
			if err = txn.Commit(); err != nil {
				err = txn.Delete(key)
			}
		}
		if err != nil {
			return
		}
	}
	return
}

func (stateMachine *StateMachine) recoverDataSize(r io.Reader) (size uint32, err error) {
	b := make([]byte, 4)
	var n int
	if n, err = io.ReadFull(r, b); err != nil {
		// 结束了
		if err == io.ErrUnexpectedEOF && n == 0 {
			err = nil
		}
		return
	}

	size = binary.BigEndian.Uint32(b)

	// size 错误
	if size <= 4 {
		err = errors.New("Size does not match")
		return
	}
	return
}

// 解析 data
func (stateMachine *StateMachine) recoverItem(r io.Reader, size uint32) (item *libraftpb.Item, err error) {
	b := make([]byte, size)

	// 读取 data hash ，data body
	if _, err = io.ReadFull(r, b); err != nil {
		return
	}

	// 验证 hash
	dataHash := binary.BigEndian.Uint32(b[0:4])
	if dataHash != crc32.ChecksumIEEE(b[4:]) {
		err = errors.New("Hash does not match")
		return
	}

	// 解码
	item = &libraftpb.Item{}
	if err = item.Unmarshal(b[4:]); err != nil {
		item = nil
		return
	}
	return
}

func (stateMachine *StateMachine) lookupEntry(txn *badger.Txn, entry *libraftpb.LookupRequestEntry, responseField *libraftpb.ResponseField) (responseData *libraftpb.ResponseData, err error) {
	// 查询
	var pbItem *libraftpb.Item
	responseData = &libraftpb.ResponseData{}
	err = stateMachine.query(txn, entry.Query, responseField, func(item *badger.Item) (err error) {
		if pbItem, err = stateMachine.toPBItem(item, responseField); err != nil {
			return
		}
		responseData.Items = append(responseData.Items, pbItem)
		return
	})
	if err != nil {
		return
	}
	return
}

func (stateMachine *StateMachine) updateEntry(txn *badger.Txn, entry *libraftpb.UpdateRequestEntry, eventBatch *StateMachineEventBatch, responseField *libraftpb.ResponseField) (responseData *libraftpb.ResponseData, err error) {
	// 不创建 更新
	if len(entry.Key) == 0 && len(entry.Value) == 0 && entry.Expire == 0 {
		return
	}

	responseData = &libraftpb.ResponseData{}
	var pbItem *libraftpb.Item

	// 无条件
	if emptyCondition(entry.Query.Key) && emptyCondition(entry.Query.Value) && entry.Query.Limit == 0 {
		// 没 key  没操作 直接返回
		if len(entry.Key) == 0 {
			return
		}

		// 读取单个 item
		var item *badger.Item
		if item, err = txn.Get(stateMachine.NewKey(entry.Key)); err != nil && err != badger.ErrKeyNotFound {
			return
		}
		err = nil

		// 不设置 value  或 删除 直接返回因为不存在
		if item == nil {
			if entry.Value == nil || entry.Expire == 1 {
				return
			}
		}

		// 修改单个
		if pbItem, err = stateMachine.updateEntryOne(txn, entry, eventBatch, item, responseField); err != nil {
			return
		}

		// 添加进响应
		if pbItem != nil {
			responseData.Items = append(responseData.Items, pbItem)
		}
		return
	}

	// 查询
	var count = 0
	err = stateMachine.query(txn, entry.Query, responseField, func(item *badger.Item) (err error) {
		// 修改 key 只能修改一个
		if len(entry.Key) != 0 && count != 0 {
			err = errors.New("Multiple items are not allowed to update")
			return
		}

		// 修改查询到的
		if pbItem, err = stateMachine.updateEntryOne(txn, entry, eventBatch, item, responseField); err != nil {
			return
		}

		if pbItem != nil {
			responseData.Items = append(responseData.Items, pbItem)
		}

		count++
		return
	})

	if err != nil {
		return
	}

	return
}

func (stateMachine *StateMachine) query(txn *badger.Txn, pbQuery *libraftpb.Query, responseField *libraftpb.ResponseField, cb func(item *badger.Item) (err error)) (err error) {

	// key 条件
	keyQuery := pbQuery.Key
	valueQuery := pbQuery.Value

	var limit uint32
	var item *badger.Item

	// eq 条件
	if len(keyQuery.Eq) != 0 {
		if pbQuery.Reverse {
			// 倒序
			for i := len(keyQuery.Eq); i >= 0; i-- {
				key := keyQuery.Eq[i]

				// key 匹配
				if !stateMachine.match(key, keyQuery) {
					continue
				}

				if item, err = txn.Get(stateMachine.NewKey(key)); err != nil {
					if err != badger.ErrKeyNotFound {
						return
					}
					err = nil
					continue
				}

				var ok bool
				if ok, err = stateMachine.valueMatch(item, valueQuery); err != nil {
					return
				}
				if !ok {
					continue
				}

				if err = cb(item); err != nil {
					return
				}
				limit++

				// 限制数量
				if pbQuery.Limit != 0 && limit > pbQuery.Limit {
					break
				}
			}
		} else {
			// 正序
			for i := 0; i < len(keyQuery.Eq); i++ {
				key := keyQuery.Eq[i]

				// key 匹配
				if !stateMachine.match(key, keyQuery) {
					continue
				}

				if item, err = txn.Get(stateMachine.NewKey(key)); err != nil {
					if err != badger.ErrKeyNotFound {
						return
					}
					err = nil
					continue
				}

				var ok bool
				if ok, err = stateMachine.valueMatch(item, valueQuery); err != nil {
					return
				}
				if !ok {
					continue
				}

				if err = cb(item); err != nil {
					return
				}
				limit++

				// 限制数量
				if pbQuery.Limit != 0 && limit > pbQuery.Limit {
					break
				}
			}
		}
		return
	}

	// 查询条件 没键入
	if len(keyQuery.St) == 0 && len(keyQuery.Ed) == 0 && pbQuery.Limit == 0 {
		err = errors.New("Query key condition is not entered")
		return
	}

	var prefetchSize = 0
	if pbQuery.Limit > 10 || pbQuery.Limit == 0 {
		prefetchSize = 10
	} else {
		prefetchSize = int(pbQuery.Limit)
	}

	it := txn.NewIterator(badger.IteratorOptions{
		Reverse:        pbQuery.Reverse,
		Prefix:         stateMachine.NewKey(keyQuery.Prefix),
		PrefetchValues: responseField.Value,
		PrefetchSize:   prefetchSize,
	})
	defer it.Close()

	// 开始
	var st []byte
	if len(keyQuery.St) != 0 {
		st = stateMachine.NewKey(keyQuery.St)
	}

	for it.Seek(st); it.Valid(); it.Next() {
		item := it.Item()

		// key 解析
		var key []byte
		if key, err = stateMachine.ParseKey(item.KeyCopy(nil)); err != nil {
			return
		}

		// 结束
		if len(keyQuery.Ed) != 0 && bytes.Compare(keyQuery.Ed, key) != 1 {
			break
		}

		// key 匹配
		if !stateMachine.match(key, keyQuery) {
			continue
		}

		if err = cb(item); err != nil {
			return
		}
		limit++

		// 限制数量
		if pbQuery.Limit != 0 && limit > pbQuery.Limit {
			break
		}
	}

	return
}

func (stateMachine *StateMachine) updateEntryOne(txn *badger.Txn, entry *libraftpb.UpdateRequestEntry, eventBatch *StateMachineEventBatch, item *badger.Item, responseField *libraftpb.ResponseField) (pbItem *libraftpb.Item, err error) {

	// 动作 匹配
	switch entry.Action {
	case libraftpb.UpdateRequestAction_CREATE:
		// 创建  存在 跳过
		if item != nil {
			return
		}
	case libraftpb.UpdateRequestAction_UPDATE:
		// 更新 不存在 返回
		if item == nil {
			return
		}
	case libraftpb.UpdateRequestAction_DELETE:
		// 删除 不存在 返回
		if item != nil {
			return
		}
	}

	var oldItem *libraftpb.Item
	var newItem *libraftpb.Item

	// 删除
	if entry.Expire == 1 || entry.Action == libraftpb.UpdateRequestAction_DELETE {
		// 不存在 直接返回
		if item == nil {
			return
		}

		// key 字段 不匹配
		if len(entry.Key) != 0 && bytes.Equal(stateMachine.NewKey(entry.Key), item.Key()) {
			err = errors.New("item does not exist")
			return
		}

		// oldItem
		if oldItem, err = stateMachine.toPBEventItem(item); err != nil {
			return
		}

		// 添加事件
		eventBatch.Add(newItem, oldItem)

		// pbItem
		if pbItem, err = stateMachine.toPBItem(item, responseField); err != nil {
			return
		}

		// 删除
		err = txn.Delete(item.KeyCopy(nil))

		return
	}

	if item != nil {
		if oldItem, err = stateMachine.toPBEventItem(item); err != nil {
			return
		}
	}

	// db 输入
	dbEntry := &badger.Entry{
		Key:       entry.Key,
		Value:     entry.Value,
		ExpiresAt: entry.Expire,
	}

	// key
	if len(dbEntry.Key) != 0 {
		dbEntry.Key = stateMachine.NewKey(dbEntry.Key)
	} else if item != nil {
		dbEntry.Key = item.KeyCopy(nil)
	} else {
		err = errors.New("entry key is empty")
		return
	}

	// value
	if len(dbEntry.Value) != 0 {
		// 设置
	} else if item != nil {
		if dbEntry.Value, err = item.ValueCopy(nil); err != nil {
			return
		}
	} else {
		err = errors.New("entry value is empty")
		return
	}

	// 过期时间
	if dbEntry.ExpiresAt == 0 && item != nil {
		dbEntry.ExpiresAt = item.ExpiresAt()
	}

	// 最大过期时间
	if dbEntry.ExpiresAt == math.MaxUint64 {
		dbEntry.ExpiresAt = 0
	}

	// 更新
	if item != nil {
		set := false
		// 是否修改 key
		if !set && !bytes.Equal(item.Key(), dbEntry.Key) {
			set = true
			// 修改 key 要删除当前 key 再添加
			if err = txn.Delete(item.KeyCopy(nil)); err != nil {
				return
			}
		}

		// 是否修改 expire
		if !set && item.ExpiresAt() != dbEntry.ExpiresAt {
			set = true
		}

		// 是否修改 value
		if !set {
			err = item.Value(func(val []byte) error {
				if !bytes.Equal(val, dbEntry.Value) {
					set = true
				}
				return nil
			})
			if err != nil {
				return
			}
		}

		// 无需更新
		if !set {
			return
		}
	}

	// 设置
	if err = txn.SetEntry(dbEntry); err != nil {
		return
	}

	// 读取新的
	if item, err = txn.Get(dbEntry.Key); err != nil {
		return
	}

	// 新的 item
	if newItem, err = stateMachine.toPBEventItem(item); err != nil {
		return
	}

	// 添加事件
	eventBatch.Add(newItem, oldItem)

	// 返回数据
	if pbItem, err = stateMachine.toPBItem(item, responseField); err != nil {
		return
	}

	return
}

func (stateMachine *StateMachine) match(val []byte, condition *libraftpb.Condition) (ok bool) {
	ok = true

	// 前缀
	if len(condition.Prefix) != 0 {
		if ok = bytes.HasPrefix(val, condition.Prefix); !ok {
			return
		}
	}

	// 包含
	if len(condition.Contains) != 0 {
		if ok = bytes.Contains(val, condition.Contains); !ok {
			return
		}
	}

	// 后缀
	if len(condition.Suffix) != 0 {
		if ok = bytes.HasSuffix(val, condition.Suffix); !ok {
			return
		}
	}

	// 相等
	if len(condition.Eq) != 0 {
		ok = false
		for _, eq := range condition.Eq {
			if bytes.Equal(eq, val) {
				ok = true
				break
			}
		}
		if !ok {
			return
		}
	}

	// 不相等
	if len(condition.Ne) != 0 {
		for _, ne := range condition.Eq {
			if bytes.Equal(ne, val) {
				ok = false
				break
			}
		}
		if !ok {
			return
		}
	}

	// st 开始 包含开始 st > val 返回 false
	if len(condition.St) != 0 {
		if bytes.Compare(condition.St, val) == 1 {
			// st 大于 val
			ok = false
			return
		}
	}

	// ed 结束 不包含结束  ed <= val 返回 false
	if len(condition.Ed) != 0 {
		if bytes.Compare(condition.Ed, val) != 1 {
			// ed 小于等于 val
			ok = false
			return
		}
	}

	return
}
func (stateMachine *StateMachine) valueMatch(item *badger.Item, condition *libraftpb.Condition) (ok bool, err error) {
	// 匹配成功
	ok = true

	//  condition 是空
	if emptyCondition(condition) {
		return
	}

	err = item.Value(func(val []byte) (err error) {
		ok = stateMachine.match(val, condition)
		return
	})

	return
}

func (stateMachine *StateMachine) NewKey(key []byte) []byte {
	l := len(stateMachineKeyPrefix)
	keyPrefix := make([]byte, l+8)
	if len(key) == 0 {
		return keyPrefix
	}
	copy(key, stateMachineKeyPrefix)
	binary.BigEndian.PutUint64(key[l:], stateMachine.clusterID)
	return bytes.Join([][]byte{keyPrefix, key}, []byte{})
}

// key 解析
func (stateMachine *StateMachine) ParseKey(key []byte) ([]byte, error) {
	l := len(stateMachineKeyPrefix)

	// 长度不匹配
	if (l + 8) >= len(key) {
		return nil, errors.New("invalid key")
	}

	// 前缀不匹配
	if !bytes.Equal(key[0:l], stateMachineKeyPrefix) {
		return nil, errors.New("invalid key")
	}

	// 集群 id 不匹配
	if binary.BigEndian.Uint64(key[l:l+8]) != stateMachine.clusterID {
		return nil, errors.New("invalid key")
	}

	return key[l+8:], nil
}

func (stateMachine *StateMachine) toPBItem(item *badger.Item, responseField *libraftpb.ResponseField) (pbItem *libraftpb.Item, err error) {
	pbItem = &libraftpb.Item{}
	if responseField.Key {
		if pbItem.Key, err = stateMachine.ParseKey(item.KeyCopy(nil)); err != nil {
			return
		}
	}

	if responseField.Value {
		if pbItem.Value, err = item.ValueCopy(nil); err != nil {
			return
		}
	}
	if responseField.Expire {
		pbItem.Expire = item.ExpiresAt()
		if pbItem.Expire == 0 {
			pbItem.Expire = math.MaxUint64
		}
	}
	return
}

func (stateMachine *StateMachine) toPBEventItem(item *badger.Item) (pbItem *libraftpb.Item, err error) {
	pbItem = &libraftpb.Item{}
	if pbItem.Key, err = stateMachine.ParseKey(item.KeyCopy(nil)); err != nil {
		return
	}
	if pbItem.Value, err = item.ValueCopy(nil); err != nil {
		return
	}
	pbItem.Expire = item.ExpiresAt()
	if pbItem.Expire == 0 {
		pbItem.Expire = math.MaxUint64
	}
	return
}

func emptyCondition(condition *libraftpb.Condition) bool {
	if condition == nil {
		return true
	}
	if len(condition.Prefix) != 0 {
		return false
	}
	if len(condition.Contains) != 0 {
		return false
	}
	if len(condition.Suffix) != 0 {
		return false
	}
	if len(condition.Eq) != 0 {
		return false
	}
	if len(condition.Ne) != 0 {
		return false
	}

	if len(condition.Ed) != 0 {
		return false
	}

	if len(condition.St) != 0 {
		return false
	}

	return true
}
