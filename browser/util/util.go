//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package util

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Set struct {
	// struct为结构体类型的变量
	M map[interface{}]struct{}
}

var Exists = struct{}{}

func DataToString(data interface{}) string {
	const MaxStringLength = 65535
	if str, ok := data.(string); ok {
		if len(str) > MaxStringLength {
			return str[0:MaxStringLength]
		}
		return str
	} else {
		return ""
	}
}
func New(items ...interface{}) *Set {
	// 获取Set的地址
	s := &Set{}
	// 声明map类型的数据结构
	s.M = make(map[interface{}]struct{})
	s.Add(items...)
	return s
}
func (s *Set) Add(items ...interface{}) error {
	if s.M == nil {
		s.M = make(map[interface{}]struct{})
	}
	for _, item := range items {
		s.M[item] = Exists
	}
	return nil
}

func ObjectTojson(ob interface{}) string {
	if ob == nil {
		return ""
	}
	result, _ := json.Marshal(ob)
	return strings.Trim(string(result), "\"")

}
func InsertUint64SliceCopy(slice, insertion []uint64, index int) []uint64 {
	result := make([]uint64, len(slice)+len(insertion))
	at := copy(result, slice[:index])
	at += copy(result[at:], insertion)
	copy(result[at:], slice[index:])
	fmt.Printf("%6T\n", at)
	return result
}
func Errors(error error) bool {
	if error != nil {
		fmt.Println("update/add error", error)
		return false
	}
	return true
}
