# HE3MapViewer

This program works with Hex Empire 3 map files, which end in the file extension .he3.

```
./HexEmpire3Map.exe -mode=[mode] -input=[input filename] -output=[output filename]
```

### File format

The .he3 map is compressed using the LZF algorithm and then encoded in base64. To read the file, the file must be decoded from base64 and decompressed using the LZF algorithm to get the raw data.

All strings consist of an integer denoting the length of the string followed by the string contents. Some of the data is optional depending on the value of the previous fields.

Map file format
```
String Format (Newer maps are always set to "hexmap")
Int32 VersionNumber (Latest version is 7. Version >= 5 will have a map style saved)
String MapTitle
String MapAuthor
Int32 Width
Int32 Depth
Byte[5] MapStyle
MapTile[Width][Depth] MapTileData
Boolean GameState (Set to false when there is no save data)
```

Map tile format
```
Float32 Height
Byte Flags (First leftmost bit is HasFlag, second leftmost bit is HasRoad, the remaining data is the tile type)
String CityName (If the tile type is in the list [airport, factory, town, city, capital], it will have a city name. For the rest of the tile types, this field is skipped.)
Int32 Party (A value between 0 and 5 inclusive, -1 means neutral)
Boolean HasInfantry
Army Infantry (Only stored if HasInfantry is true)
Boolean HasArtillery
Army Artillery (Only stored if HasArtillery is true)
```

Army format
```
Int32 X
Int32 Y
Int32 UnitInfantry 
Int32 UnitArtillery
Float32 Morale
```

### Visualize map

Convert Hex Empire 3 maps to a PNG file so that you can visualize the map layout in a single image.

Example
```
./HexEmpire3Map.exe -mode=visualize -input=map.he3 -output=map.png
```

<div style="display:inline-block;">
<img src="https://raw.githubusercontent.com/samuelyuan/HexEmpire3Map/master/screenshots/tropic-of-cancer.png" alt="map" width="400" height="400" />
</div>

### Decompress

The .he3 maps are compressed, but if you decompress the map, you can better understand the file structure and make changes
to the maps more easily without using the map editor.

Example
```
./HexEmpire3Map.exe -mode=decompress -input=map.he3 -output=decompressed_map.he3decomp
```

### Compress

You can take a decompressed map and compress it again so that it can be recognized by this tool and the original game.

Example
```
./HexEmpire3Map.exe -mode=compress -input=decompressed_map.he3decomp -output=compressed_map.he3
```
