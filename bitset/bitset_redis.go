package bitset

import (
	"bytes"
	"context"
	"fmt"

	"encoding/base64"
	"encoding/binary"
	"encoding/json"

	"github.com/gostatix"
	"github.com/redis/go-redis/v9"
)

type BitSetRedis struct {
	size uint
	key  string
}

func NewBitSetRedis(size uint, key string) *BitSetRedis {
	return &BitSetRedis{size, key}
}

func FromDataRedis(data []uint64, key string) (*BitSetRedis, error) {
	bitSetRedis := NewBitSetRedis(uint(len(data)), key)
	bytes, err := uint64ArrayToByteArray(data)
	if err != nil {
		return nil, err
	}
	_, err = bitSetRedis.Import(bitSetRedis.size, bytes)
	if err != nil {
		return nil, err
	}
	return bitSetRedis, nil
}

func (bitSet BitSetRedis) Has(index uint) (bool, error) {
	val, err := gostatix.GetRedisClient().GetBit(context.Background(), bitSet.key, int64(index)).Result()
	if err != nil {
		return false, err
	}
	return val != 0, nil
}

func (bitSet BitSetRedis) Insert(index uint) (bool, error) {
	err := gostatix.GetRedisClient().SetBit(context.Background(), bitSet.key, int64(index), 1).Err()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (aSet BitSetRedis) Equals(otherBitSet IBitSet) (bool, error) {
	bSet, ok := otherBitSet.(*BitSetRedis)
	if !ok {
		return false, fmt.Errorf("invalid bitset type, should be BitSetRedis")
	}
	aSetVal, err1 := gostatix.GetRedisClient().Get(context.Background(), aSet.key).Result()
	if err1 != nil {
		return false, err1
	}
	bSetVal, err2 := gostatix.GetRedisClient().Get(context.Background(), bSet.key).Result()
	if err2 != nil {
		return false, err2
	}
	return aSetVal == bSetVal, nil
}

func (bitSet BitSetRedis) Max() (uint, bool) {
	index, err := gostatix.GetRedisClient().BitPos(context.Background(), bitSet.key, 1).Result()
	if err != nil || index == -1 {
		return 0, false
	}
	return uint(index), true
}

func (bitSet BitSetRedis) BitCount() (uint, error) {
	bitRange := &redis.BitCount{Start: 0, End: -1}
	val, err := gostatix.GetRedisClient().BitCount(context.Background(), bitSet.key, bitRange).Result()
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}

func (bitSet BitSetRedis) Export() (uint, []byte, error) {
	val, err := gostatix.GetRedisClient().Get(context.Background(), bitSet.key).Result()
	if err != nil {
		return 0, nil, err
	}
	data, err := json.Marshal(base64.URLEncoding.EncodeToString([]byte(val)))
	if err != nil {
		return 0, nil, err
	}
	return bitSet.size, data, nil
}

func (bitSet BitSetRedis) Import(size uint, data []byte) (bool, error) {
	var bytes []byte
	err := json.Unmarshal(data, &bytes)
	if err != nil {
		return false, err
	}
	_, err = base64.URLEncoding.Decode(bytes, bytes)
	if err != nil {
		return false, err
	}
	err = gostatix.GetRedisClient().Set(context.Background(), bitSet.key, string(bytes), 0).Err()
	if err != nil {
		return false, err
	}
	return true, nil
}

func uint64ArrayToByteArray(data []uint64) ([]byte, error) {
	// Create a buffer to store the bytes
	buf := new(bytes.Buffer)

	// Write each uint64 element to the buffer
	for _, v := range data {
		if err := binary.Write(buf, binary.BigEndian, v); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
