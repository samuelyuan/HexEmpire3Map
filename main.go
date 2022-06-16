package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/fogleman/gg"
	"github.com/samuelyuan/HexEmpire3Map/fileio"
)

type MapData struct {
	MapTiles [][]*fileio.MapTile
	Width    int
	Depth    int
}

var (
	NeighborOdd  = [6][2]int{{-1, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}}
	NeighborEven = [6][2]int{{-1, 0}, {-1, -1}, {0, -1}, {1, 0}, {0, 1}, {-1, 1}}
	PartyColors  = [6][3]int{{0, 76, 229}, {178, 0, 204}, {255, 8, 8}, {0, 153, 0}, {204, 127, 0}, {0, 127, 115}}
)

func readData(filename string) (*MapData, error) {
	mapTiles := fileio.ReadHE3File(filename)

	mapData := &MapData{
		MapTiles: mapTiles,
		Width:    len(mapTiles),
		Depth:    len(mapTiles[0]),
	}
	return mapData, nil
}

func getNeighbors(x int, z int) [6][2]int {
	var offset [6][2]int
	if z%2 == 1 {
		offset = NeighborOdd
	} else {
		offset = NeighborEven
	}

	neighbors := [6][2]int{}
	for i := 0; i < 6; i++ {
		newX := x + offset[i][0]
		newZ := z + offset[i][1]
		neighbors[i][0] = newX
		neighbors[i][1] = newZ
	}
	return neighbors
}

func isPort(x int, z int, mapData *MapData) bool {
	if mapData.MapTiles[x][z].TileType < fileio.Town {
		return false
	}

	neighbors := getNeighbors(x, z)
	for i := 0; i < len(neighbors); i++ {
		newX := neighbors[i][0]
		newZ := neighbors[i][1]
		if newX >= 0 && newZ >= 0 && newX < mapData.Width && newZ < mapData.Depth {
			if mapData.MapTiles[newX][newZ].IsSea {
				return true
			}
		}
	}
	return false
}

func getImagePosition(i int, j int) (float64, float64) {
	radius := 10.0
	angle := math.Pi / 6

	x := (radius * 1.5) + float64(j)*(2*radius*math.Cos(angle))
	y := radius + float64(i)*radius*(1+math.Sin(angle))
	if i%2 == 1 {
		x += radius * math.Cos(angle)
	}
	return x, y
}

func removeAccents(str string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	newStr, _, err := transform.String(t, str)
	if err != nil {
		log.Fatal("Error removing accents from string "+str+":", err)
	}
	return newStr
}

func drawMap(mapData *MapData, outputFilename string) {
	radius := 10.0

	mapWidth := len(mapData.MapTiles)
	mapDepth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := getImagePosition(mapDepth, mapWidth)
	dc := gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	fmt.Println("Map depth: ", mapDepth, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	dc.InvertY()

	for i := 0; i < mapData.Depth; i++ {
		for j := 0; j < mapData.Width; j++ {
			x, y := getImagePosition(i, j)
			dc.DrawRegularPolygon(6, x, y, radius, math.Pi/2)

			tile := mapData.MapTiles[j][i]
			if tile.IsSea {
				dc.SetRGB255(95, 149, 149)
			} else if isPort(j, i, mapData) {
				dc.SetRGB255(75, 113, 224)
			} else {
				switch tile.TileType {
				case fileio.Grass:
					dc.SetRGB255(105, 125, 54)
				case fileio.Sand:
					dc.SetRGB255(200, 200, 164)
				case fileio.Farmland:
					dc.SetRGB255(127, 121, 71)
				case fileio.Forest:
					dc.SetRGB255(53, 72, 44)
				case fileio.Snow:
					dc.SetRGB255(238, 249, 255)
				case fileio.Factory:
					dc.SetRGB255(213, 95, 7)
				case fileio.Town:
					dc.SetRGB255(1, 137, 26)
				case fileio.City:
					dc.SetRGB255(87, 88, 80)
				default:
					dc.SetRGB255(0, 0, 0)
				}
			}

			dc.Fill()

			if tile.IsMountain {
				dc.DrawRegularPolygon(3, x, y, radius, math.Pi)
				dc.SetRGB255(89, 90, 86)
				dc.Fill()
				dc.DrawRegularPolygon(3, x, y+(radius/2), radius/2, math.Pi)
				dc.SetRGB255(234, 244, 253)
				dc.Fill()
			}

			if tile.TileType == fileio.Factory ||
				tile.TileType == fileio.City ||
				tile.TileType == fileio.Town {
				if tile.HasFlag && tile.Party >= 0 {
					// Draw capital city
					dc.DrawCircle(x, y, radius/2)
					dc.SetRGB255(PartyColors[tile.Party][0], PartyColors[tile.Party][1], PartyColors[tile.Party][2])
					dc.Fill()
				} else {
					dc.DrawRectangle(x-2.0, y-2.0, radius/2, radius/2)
					dc.SetRGB255(255, 255, 255)
					dc.Fill()
				}
			}
		}
	}

	// Draw roads between tiles
	for i := 0; i < mapData.Depth; i++ {
		for j := 0; j < mapData.Width; j++ {
			x1, y1 := getImagePosition(i, j)

			if !mapData.MapTiles[j][i].HasRoad {
				continue
			}

			neighbors := getNeighbors(j, i)
			for n := 0; n < len(neighbors); n++ {
				newX := neighbors[n][0]
				newZ := neighbors[n][1]
				if newX >= 0 && newZ >= 0 && newX < mapData.Width && newZ < mapData.Depth {
					if mapData.MapTiles[newX][newZ].HasRoad || mapData.MapTiles[newX][newZ].TileType >= fileio.Factory {
						x2, y2 := getImagePosition(newZ, newX)
						dc.SetRGB255(78, 53, 36)
						dc.DrawLine(x1, y1, x2, y2)
						dc.Stroke()
					}
				}
			}
		}
	}

	// Draw city names on top of hexes
	dc.InvertY()
	for i := 0; i < mapData.Depth; i++ {
		for j := 0; j < mapData.Width; j++ {
			// Invert depth because the map is inverted
			x, y := getImagePosition(mapData.Depth-i, j)

			tile := mapData.MapTiles[j][i]
			dc.SetRGB255(255, 255, 255)
			dc.DrawString(removeAccents(tile.CityName), x-(5.0*float64(len(tile.CityName))/2.0), y-radius*1.5)
		}
	}

	dc.SavePNG(outputFilename)
	fmt.Println("Saved image to", outputFilename)
}

func main() {
	availableModes := "[visualize, decompress, compress]"
	modePtr := flag.String("mode", "", "Available modes: "+availableModes)
	inputPtr := flag.String("input", "", "Input filename")
	outputPtr := flag.String("output", "output.png", "Output filename")
	flag.Parse()

	mode := *modePtr
	inputFilename := *inputPtr
	outputFilename := *outputPtr
	fmt.Println("Mode: ", mode)
	fmt.Println("Input filename: ", inputFilename)
	fmt.Println("Output filename: ", outputFilename)

	if mode == "visualize" {
		mapData, err := readData(inputFilename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}
		drawMap(mapData, *outputPtr)
	} else if mode == "decompress" {
		decompressedBytes := fileio.DecompressHE3File(inputFilename)
		err := os.WriteFile(outputFilename, decompressedBytes, 0644)
		if err != nil {
			log.Fatal("Failed to write to output file: ", err)
		}
	} else if mode == "compress" {
		decompressedBytes, err := ioutil.ReadFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}
		compressedData := base64.StdEncoding.EncodeToString(fileio.Compress(decompressedBytes))
		err = os.WriteFile(outputFilename, []byte(compressedData), 0644)
		if err != nil {
			log.Fatal("Failed to write to output file: ", err)
		}
	} else {
		log.Fatal("Invalid mode. One of the following modes are supported " + availableModes)
	}
}
