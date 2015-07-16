package main

import (
	"fmt"
	"log"

	"github.com/jonas-p/go-shp"
	"github.com/kellydunn/golang-geo"
	gj "github.com/kpawlik/geojson"
)

// Tracker Struct
type Tracker struct {
	Pipes           []shp.PolyLine
	Points          []geo.Point
	Segments        []Segment
	Connections     [][]int
	PipeCount       int
	PointCount      int
	SegmentCount    int
	ConnectionCount int
}

// Segment is a connection between 2 points
type Segment struct {
	start   *geo.Point
	startid int
	end     *geo.Point
	endid   int
}

//Open Shape File
func (t *Tracker) Open(file string) {

	fmt.Println("Opened: ", file)
	fmt.Println("Reading File")

	shape, err := shp.Open(file)
	if err != nil {
		log.Fatal(err)
	}

	for shape.Next() {
		_, p := shape.Shape()

		pl, _ := p.(*shp.PolyLine)
		t.Pipes = append(t.Pipes, *pl)

	}
	shape.Close()
	fmt.Println("Closed File")

	t.PipeCount = len(t.Pipes)
	fmt.Println("t.generatePointsAndSegments()")
	t.generatePointsAndSegments()
	fmt.Println("point count after gen: ", len(t.Points))
	fmt.Println("t.createConnectionList()")
	t.createConnectionList()
	for _, c := range t.Connections {
		if len(c) > 1 {
			fmt.Println(c)
		}
	}
}

func (t *Tracker) generatePointsAndSegments() {
	var frompointid int
	var topointid int

	for _, pipe := range t.Pipes {

		//for each pipe i.e PolyLine
		for n, point := range pipe.Points {
			//create new geo.Point
			geopoint := geo.NewPoint(point.Y, point.X)

			//see if the point already exists
			var found bool
			found = false
			for existingpointid, existingpoint := range t.Points {
				if geopoint.Lat() == existingpoint.Lat() && geopoint.Lng() == existingpoint.Lat() {
					topointid = existingpointid

					found = true
				}
			}
			if !found {
				t.Points = append(t.Points, *geopoint)
				topointid = len(t.Points) - 1
			}

			//fmt.Println("point: ", n)
			//fmt.Println("fromid: ", frompointid)
			//	fmt.Println("topointid:", topointid)

			if n != 0 {
				seg := new(Segment)
				seg.start = &t.Points[frompointid]
				seg.startid = frompointid
				seg.end = &t.Points[topointid]
				seg.endid = topointid
				t.Segments = append(t.Segments, *seg)
			}

			frompointid = topointid
		}
	}
	t.PointCount = len(t.Points)
	t.SegmentCount = len(t.Segments)

	for n, seg := range t.Segments {
		fmt.Println(seg)
		if n > 10 {
			break
		}
	}
}

func (t *Tracker) createConnectionList() {
	var count int
	t.Connections = make([][]int, t.PointCount)
	// fmt.Println(len(t.Connections))
	//fmt.Println(t.SegmentCount)

	for _, segment := range t.Segments {
		from := segment.startid
		to := segment.endid

		t.Connections[from] = append(t.Connections[from], to)
		count = count + 1
	}
	t.ConnectionCount = count
	fmt.Println("ConnectionCount = ", t.ConnectionCount)

	for n, con := range t.Connections {
		fmt.Println(con)
		if n > 10 {
			break
		}
	}
	//fmt.Println(t.Connections)
}

// FindPath return geo.Points of path nearest to input point
func (t Tracker) FindPath(point int) (path []geo.Point, found bool) {
	found = false
	path = append(path, t.Points[point])
	for len(t.Connections[point]) > 0 {
		path = append(path, t.Points[t.Connections[point][0]])
		point = t.Connections[point][0]
		found = true
	}
	return path, found
}

// FindNearestPoint
// input *geo.Point
// minimum float64: exclude points closer than this distance
//returns id, *geo.Point, distance to point

func (t *Tracker) FindNearestPoint(input *geo.Point, minimum float64) (int, *geo.Point, float64) {
	//fmt.Println("in FindNearestPoint")

	var closestdistance float64
	var nearestpoint *geo.Point
	var nearestpointid int
	closestdistance = 7000000
	for id, point := range t.Points {
		distance := input.GreatCircleDistance(&point)
		if distance < closestdistance && distance > minimum {
			closestdistance = distance
			nearestpoint = &point
			nearestpointid = id
		}

	}
	//fmt.Println("nearestpointid: ", nearestpointid)
	//fmt.Println("nearestpoint:, ", nearestpoint)
	return nearestpointid, nearestpoint, closestdistance
}

// PathToGeoJSON ..
func (t Tracker) PathToGeoJSON(path []geo.Point) (string, error) {
	fc := new(gj.FeatureCollection)
	var f *gj.Feature
	line := new(gj.LineString)
	line.Type = `LineString`
	//var cordinates gj.Coordinates

	for _, p := range path {
		coord := new(gj.Coordinate)
		coord[0] = gj.Coord(p.Lng())
		coord[1] = gj.Coord(p.Lat())
		line.AddCoordinates(*coord)
		//cordinates = append(cordinates, *coord)
	}
	//line.AddCoordinates(cordinates)

	fmt.Println(line.Type)
	f = gj.NewFeature(line, nil, nil)
	fc.AddFeatures(f)
	fc.Type = `FeatureCollection`
	geojson, e := gj.Marshal(fc)
	return geojson, e
}
