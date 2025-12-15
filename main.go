package main

import (
	"encoding/base64"
	"flag"
	"fmt"
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

const (
	HexRadius = 10.0
)

var (
	NeighborOdd  = [6][2]int{{-1, 0}, {0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}}
	NeighborEven = [6][2]int{{-1, 0}, {-1, -1}, {0, -1}, {1, 0}, {0, 1}, {-1, 1}}
	// PartyColors represents the colors for each faction
	PartyColors = [6][3]int{
		{0, 76, 229},   // Party 0: Bluegaria (Blue)
		{178, 0, 204},  // Party 1: Violetnam (Purple)
		{255, 8, 8},    // Party 2: Redosia (Red)
		{0, 153, 0},    // Party 3: Greenland (Green)
		{204, 127, 0},  // Party 4: Amberica (Amber)
		{0, 127, 115},  // Party 5: Turquoistan (Turquoise)
	}
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
	radius := HexRadius
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

func getTileColor(tile *fileio.MapTile, mapData *MapData, x, z int) (r, g, b int) {
	if tile.IsSea {
		return 95, 149, 149
	} else if isPort(x, z, mapData) {
		return 75, 113, 224
	} else {
		switch tile.TileType {
		case fileio.Grass:
			return 105, 125, 54
		case fileio.Sand:
			return 200, 200, 164
		case fileio.Farmland:
			return 127, 121, 71
		case fileio.Forest:
			return 53, 72, 44
		case fileio.Snow:
			return 238, 249, 255
		case fileio.Factory:
			return 213, 95, 7
		case fileio.Town:
			return 1, 137, 26
		case fileio.City:
			return 87, 88, 80
		default:
			return 0, 0, 0
		}
	}
}

func drawMountain(dc *gg.Context, x, y float64) {
	dc.DrawRegularPolygon(3, x, y, HexRadius, math.Pi)
	dc.SetRGB255(89, 90, 86)
	dc.Fill()
	dc.DrawRegularPolygon(3, x, y+(HexRadius/2), HexRadius/2, math.Pi)
	dc.SetRGB255(234, 244, 253)
	dc.Fill()
}

func drawCityMarker(dc *gg.Context, x, y float64, tile *fileio.MapTile) {
	if tile.HasFlag && tile.Party >= 0 {
		// Draw capital city
		dc.DrawCircle(x, y, HexRadius/2)
		dc.SetRGB255(PartyColors[tile.Party][0], PartyColors[tile.Party][1], PartyColors[tile.Party][2])
		dc.Fill()
	} else {
		dc.DrawRectangle(x-2.0, y-2.0, HexRadius/2, HexRadius/2)
		dc.SetRGB255(255, 255, 255)
		dc.Fill()
	}
}

func drawTiles(dc *gg.Context, mapData *MapData) {
	for i := 0; i < mapData.Depth; i++ {
		for j := 0; j < mapData.Width; j++ {
			x, y := getImagePosition(i, j)
			dc.DrawRegularPolygon(6, x, y, HexRadius, math.Pi/2)

			tile := mapData.MapTiles[j][i]
			r, g, b := getTileColor(tile, mapData, j, i)
			dc.SetRGB255(r, g, b)
			dc.Fill()

			if tile.IsMountain {
				drawMountain(dc, x, y)
			}

			if tile.TileType == fileio.Factory ||
				tile.TileType == fileio.City ||
				tile.TileType == fileio.Town {
				drawCityMarker(dc, x, y, tile)
			}
		}
	}
}

func isValidNeighbor(x, z, width, depth int) bool {
	return x >= 0 && z >= 0 && x < width && z < depth
}

func shouldDrawRoad(tile *fileio.MapTile) bool {
	return tile.HasRoad || tile.TileType >= fileio.Factory
}

func drawRoads(dc *gg.Context, mapData *MapData) {
	for i := 0; i < mapData.Depth; i++ {
		for j := 0; j < mapData.Width; j++ {
			if !mapData.MapTiles[j][i].HasRoad {
				continue
			}

			x1, y1 := getImagePosition(i, j)
			neighbors := getNeighbors(j, i)
			for n := 0; n < len(neighbors); n++ {
				newX := neighbors[n][0]
				newZ := neighbors[n][1]
				if isValidNeighbor(newX, newZ, mapData.Width, mapData.Depth) {
					neighborTile := mapData.MapTiles[newX][newZ]
					if shouldDrawRoad(neighborTile) {
						x2, y2 := getImagePosition(newZ, newX)
						dc.SetRGB255(78, 53, 36)
						dc.DrawLine(x1, y1, x2, y2)
						dc.Stroke()
					}
				}
			}
		}
	}
}

func drawCityNames(dc *gg.Context, mapData *MapData) {
	dc.InvertY()
	for i := 0; i < mapData.Depth; i++ {
		for j := 0; j < mapData.Width; j++ {
			// Invert depth because the map is inverted
			x, y := getImagePosition(mapData.Depth-i, j)
			tile := mapData.MapTiles[j][i]
			dc.SetRGB255(255, 255, 255)
			dc.DrawString(removeAccents(tile.CityName), x-(5.0*float64(len(tile.CityName))/2.0), y-HexRadius*1.5)
		}
	}
}

func drawMap(mapData *MapData, outputFilename string) {
	mapWidth := len(mapData.MapTiles)
	mapDepth := len(mapData.MapTiles[0])

	maxImageWidth, maxImageHeight := getImagePosition(mapDepth, mapWidth)
	dc := gg.NewContext(int(maxImageWidth), int(maxImageHeight))
	fmt.Println("Map depth: ", mapDepth, ", width: ", mapWidth)

	// Need to invert image because the map format is inverted
	dc.InvertY()

	drawTiles(dc, mapData)
	drawRoads(dc, mapData)
	drawCityNames(dc, mapData)

	dc.SavePNG(outputFilename)
	fmt.Println("Saved image to", outputFilename)
}

func printHelp() {
	fmt.Println("HexEmpire3Map - Hex Empire 3 Map Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  hexmap -mode=<mode> -input=<input> [-output=<output>]")
	fmt.Println()
	fmt.Println("Modes:")
	fmt.Println("  visualize  - Convert .he3 map file to PNG image")
	fmt.Println("  decompress - Decompress .he3 file to binary data")
	fmt.Println("  compress   - Compress binary data to .he3 format")
	fmt.Println("  help       - Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  hexmap -mode=visualize -input=maps/Europe.he3 -output=europe.png")
	fmt.Println("  hexmap -mode=decompress -input=maps/Europe.he3 -output=europe.bin")
	fmt.Println("  hexmap -mode=compress -input=europe.bin -output=europe_new.he3")
	fmt.Println()
}

func main() {
	availableModes := "[visualize, decompress, compress, help]"
	modePtr := flag.String("mode", "", "Available modes: "+availableModes)
	inputPtr := flag.String("input", "", "Input filename")
	outputPtr := flag.String("output", "output.png", "Output filename")
	flag.Parse()

	mode := *modePtr
	inputFilename := *inputPtr
	outputFilename := *outputPtr

	if mode == "help" || mode == "" {
		printHelp()
		if mode == "" {
			os.Exit(1)
		}
		return
	}

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
		decompressedBytes, err := os.ReadFile(inputFilename)
		if err != nil {
			log.Fatal("Failed to read input file: ", err)
		}
		compressedData := base64.StdEncoding.EncodeToString(fileio.Compress(decompressedBytes))
		err = os.WriteFile(outputFilename, []byte(compressedData), 0644)
		if err != nil {
			log.Fatal("Failed to write to output file: ", err)
		}
	} else {
		fmt.Println("Invalid mode. One of the following modes are supported " + availableModes)
		fmt.Println("Use -mode=help for usage information")
		os.Exit(1)
	}
}
