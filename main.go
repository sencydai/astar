package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

const (
	StraightWeight int = 10 //直线权值
	DiagonalWeight int = 14 //斜线权值
)

type MapData struct {
	Width  int
	Height int
	blocks map[int]bool
}

type BlockData struct {
	startX int //从第0行开始
	startY int //从第0列开始
	endX   int
	endY   int
}

type Path struct {
	x, y   int
	index  int
	gValue int //起点到当前点
	hValue int //当前点到目标点
	fValue int //gValue + hValue
	parent *Path
}

type Record struct {
	openMap                    map[int]*Path
	closeMap                   map[int]*Path
	startX, startY, startIndex int
	endX, endY, endIndex       int
	cur                        *Path
	selectCount                int
}

var (
	mapFile  = flag.String("m", "", "map data file")
	findPath = flag.String("p", "", "startX,startY,endX,endY")
)

func AbsInt(x int) int {
	if x >= 0 {
		return x
	}

	return -x
}

func NewMapData(width, height int, blocks ...*BlockData) *MapData {
	mapData := &MapData{width, height, make(map[int]bool, 0)}
	for _, v := range blocks {
		for i := v.startX; i <= v.endX; i++ {
			for j := v.startY; j <= v.endY; j++ {
				mapData.blocks[j*width+i] = true
			}
		}
	}

	return mapData
}

func (mapData *MapData) GetIndex(x, y int) int {
	return y*mapData.Width + x
}

func (mapData *MapData) IsInBlock(index int) bool {
	return mapData.blocks[index]
}

func (mapData *MapData) TryJoinOpenPath(record *Record, x, y, weight int) (index int) {
	index = y*mapData.Width + x
	if _, ok := mapData.blocks[index]; ok {
		return -1
	}
	if _, ok := record.closeMap[index]; ok {
		return
	}
	gValue := record.cur.gValue + weight

	if path, ok := record.openMap[index]; ok {
		if path.gValue > gValue {
			path.gValue = gValue
			path.fValue = gValue + path.hValue
			path.parent = record.cur
		}
	} else {
		path := &Path{x: x, y: y, index: index, gValue: gValue, parent: record.cur}
		path.hValue = (AbsInt(x-record.endX) + AbsInt(y-record.endY)) * StraightWeight
		path.fValue = gValue + path.hValue
		record.openMap[index] = path
	}

	return
}

func (mapData *MapData) GetOpenPath(record *Record) {
	cur := record.cur

	if cur == nil || cur.index == record.endIndex {
		return
	}

	delete(record.openMap, cur.index)
	record.closeMap[cur.index] = cur

	//左
	var left = -1
	if cur.x > 0 {
		left = mapData.TryJoinOpenPath(record, cur.x-1, cur.y, StraightWeight)
	}

	//右
	var right = -1
	if cur.x < mapData.Width-1 {
		right = mapData.TryJoinOpenPath(record, cur.x+1, cur.y, StraightWeight)
	}

	//上
	if cur.y > 0 {
		y := cur.y - 1

		//上
		if mapData.TryJoinOpenPath(record, cur.x, y, StraightWeight) != -1 {
			//左上
			if left != -1 && cur.x > 0 {
				mapData.TryJoinOpenPath(record, cur.x-1, y, DiagonalWeight)
			}

			//右上
			if right != -1 && cur.x < mapData.Width-1 {
				mapData.TryJoinOpenPath(record, cur.x+1, y, DiagonalWeight)
			}
		}

	}

	//下
	if cur.y < mapData.Height-1 {
		y := cur.y + 1

		//下
		if mapData.TryJoinOpenPath(record, cur.x, y, StraightWeight) != -1 {
			//左下
			if left != -1 && cur.x > 0 {
				mapData.TryJoinOpenPath(record, cur.x-1, y, DiagonalWeight)
			}

			//右下
			if right != -1 && cur.x < mapData.Width-1 {
				mapData.TryJoinOpenPath(record, cur.x+1, y, DiagonalWeight)
			}
		}
	}

	var path *Path
	openMap := record.openMap
	for _, item := range openMap {
		if path == nil || path.fValue > item.fValue {
			path = item
		} else if path.fValue == item.fValue && path.hValue > item.hValue {
			path = item
		}
	}

	record.cur = path
	record.selectCount++

	mapData.GetOpenPath(record)
}

func (mapData *MapData) FindingPath(startX, startY, endX, endY int) *Record {
	startIndex := mapData.GetIndex(startX, startY)
	endIndex := mapData.GetIndex(endX, endY)
	record := &Record{openMap: make(map[int]*Path), closeMap: make(map[int]*Path)}
	record.startX, record.startY, record.startIndex = startX, startY, startIndex
	record.endX, record.endY, record.endIndex = endX, endY, endIndex
	record.cur = &Path{x: startX, y: startY, index: startIndex, gValue: 0}
	mapData.GetOpenPath(record)
	return record
}

func printPath(mapData *MapData, record *Record, paths map[int]*Path) {

	//	|-----|-----|-----|-----|-----|
	//	|/////|     |     |     |  E  |
	//	|-----|-----|-----|-----|-----|
	//	|     |     |/////|     |     |
	//	|-----|-----|-----|-----|-----|
	//	|  S  |     |     |     |     |
	//	|-----|-----|-----|-----|-----|
	//
	printRows(mapData.Width)
	for col := 0; col < mapData.Height; col++ {
		printFooter(mapData.Width)
		for row := 0; row < mapData.Width; row++ {
			index := mapData.GetIndex(row, col)
			if mapData.IsInBlock(index) {
				fmt.Print("\x1b[32m|\x1b[0m")
				fmt.Print("/////")
			} else {
				// fmt.Print("\x1b[32m|\x1b[0m")
				// if index == record.startIndex {
				// 	fmt.Print("\x1b[31mStart\x1b[0m")
				// } else if index == record.endIndex {
				// 	fmt.Print(" \x1b[31mEnd\x1b[0m ")
				// } else if path, ok := paths[index]; ok {
				// 	fmt.Printf("\x1b[31m%-5d\x1b[0m", path.fValue)
				// } else if path, ok := record.closeMap[index]; ok {
				// 	fmt.Printf("%-5d", path.fValue)
				// } else {
				// 	fmt.Print("     ")
				// }

				fmt.Print("\x1b[32m|\x1b[0m")
				if index == record.startIndex {
					fmt.Print("\x1b[31mStart\x1b[0m")
				} else if index == record.endIndex {
					fmt.Print(" \x1b[31mEnd\x1b[0m ")
				} else if path, ok := paths[index]; ok {
					fmt.Printf("\x1b[31m%-5d\x1b[0m", path.fValue)
				} else if path, ok := record.closeMap[index]; ok {
					fmt.Printf("%-5d", path.fValue)
				} else {
					fmt.Print("     ")
				}
			}
		}
		fmt.Printf("\x1b[32m|\x1b[0m%d\n", col)
	}

	printFooter(mapData.Width)
	printRows(mapData.Width)
}

func printFooter(width int) {
	for i := 0; i < width; i++ {
		fmt.Print("\x1b[32m|-----\x1b[0m")
	}
	fmt.Println("\x1b[32m|\x1b[0m")
}

func printRows(width int) {
	fmt.Print(" ")
	for i := 0; i < width; i++ {
		fmt.Printf("%-5d ", i)
	}
	fmt.Println()
}

func main() {
	flag.Parse()
	if *mapFile == "" || *findPath == "" {
		flag.Usage()
		return
	}

	posInfo := strings.Split(*findPath, ",")
	if len(posInfo) != 4 {
		flag.Usage()
		return
	}

	startX, err := strconv.Atoi(posInfo[0])
	if err != nil || startX < 0 {
		flag.Usage()
		return
	}

	startY, err := strconv.Atoi(posInfo[1])
	if err != nil || startY < 0 {
		flag.Usage()
		return
	}

	endX, err := strconv.Atoi(posInfo[2])
	if err != nil || endX < 0 {
		flag.Usage()
		return
	}

	endY, err := strconv.Atoi(posInfo[3])
	if err != nil || endY < 0 {
		flag.Usage()
		return
	}

	buff, err := ioutil.ReadFile(*mapFile)
	if err != nil {
		fmt.Printf("open map data file failed: %s", err.Error())
		return
	}

	jData := make(map[string]interface{})
	if err := json.Unmarshal(buff, &jData); err != nil {
		fmt.Printf("read map data failed: %s", err.Error())
		return
	}

	width := int(jData["width"].(float64))
	high := int(jData["high"].(float64))

	blocks := make([]*BlockData, 0)
	for _, blockBuff := range jData["blocks"].([]interface{}) {
		arr := make([]int, 4)
		for i, value := range blockBuff.([]interface{}) {
			arr[i] = int(value.(float64))
		}
		block := &BlockData{}
		block.startX, block.startY, block.endX, block.endY = arr[0], arr[1], arr[2], arr[3]
		blocks = append(blocks, block)
	}

	mapData := NewMapData(width, high, blocks...)

	if startX < 0 || startX >= width || startY < 0 || startY >= high || mapData.IsInBlock(mapData.GetIndex(startX, startY)) {
		fmt.Println("start pos error")
		return
	}

	if endX < 0 || endX >= width || endY < 0 || endY >= high || mapData.IsInBlock(mapData.GetIndex(endX, endY)) {
		fmt.Println("end pos error")
		return
	}

	start := time.Now()
	record := mapData.FindingPath(startX, startY, endX, endY)

	paths := make(map[int]*Path)
	for path := record.cur; path != nil; path = path.parent {
		paths[path.index] = path
	}

	fmt.Printf("cost:%v,step:%d,selectCount(%d)\n", time.Since(start), len(paths), record.selectCount)

	printPath(mapData, record, paths)
}
