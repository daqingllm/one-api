package cache

import (
	"fmt"
	"github.com/coocood/freecache"
	"time"
)

func main() {
	// 创建一个缓存实例，分配 100MB 的内存
	cacheSize := 100 * 1024 * 1024
	cache := freecache.NewCache(cacheSize)

	// 设置缓存项，过期时间为 60 秒
	key := []byte("myKey")
	value := []byte("myValue")
	expire := 1 // 过期时间，单位为秒
	err := cache.Set(key, value, expire)
	if err != nil {
		fmt.Println("Error setting cache:", err)
		return
	}

	// 获取缓存项
	got, err := cache.Get(key)
	if err != nil {
		fmt.Println("Error getting cache:", err)
	} else {
		fmt.Println("Got value:", string(got))
	}

	time.Sleep(2 * time.Second)
	got, err = cache.Get(key)
	if err != nil {
		fmt.Println("Error getting cache:", err)
	} else {
		fmt.Println("Got value:", string(got))
	}

	// 删除缓存项
	affected := cache.Del(key)
	if affected {
		fmt.Println("Key deleted")
	} else {
		fmt.Println("Key not found")
	}

	// 尝试获取已删除的缓存项
	got, err = cache.Get(key)
	if err != nil {
		fmt.Println("Error getting cache:", err)
	} else {
		fmt.Println("Got value:", string(got))
	}

	// 获取缓存统计信息
	fmt.Println("Entry count:", cache.EntryCount())
	fmt.Println("Hit count:", cache.HitCount())
	fmt.Println("Miss count:", cache.MissCount())
	fmt.Println("Average access time:", cache.AverageAccessTime())
}
