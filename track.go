package main

import (
	"fmt"
	"github.com/jonas-p/go-shp"
	"github.com/kellydunn/golang-geo"
	gj "github.com/kpawlik/geojson"
	"io/ioutil"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	//"time"
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
	pipeid  int
}

// OpenFolder open folder of shapefiles
func (t *Tracker) OpenFolder(folder string) (err error) {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return fmt.Errorf("Opening Folder Failed")
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".shp" {
			fmt.Println("reading: ", file.Name())
			t.Open(file.Name())

		}
	}

	return err
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

	/*for _, c := range t.Connections {
		if len(c) > 1 {
			fmt.Println(c)
		}
	}*/
}

// findExisting, given a point try to find existing points concurrently
func (t *Tracker) findExisting(newpoint geo.Point) (exists bool, id int) {
	numpoints := len(t.Points)

	var NCPU int
	NCPU = runtime.NumCPU()

	found := make(chan int, 4)
	quit := make(chan bool, 1)

	if numpoints < 100 {
		for existingpointid, existingpoint := range t.Points {
			if newpoint.Lat() == existingpoint.Lat() && newpoint.Lng() == existingpoint.Lng() {
				return true, existingpointid
			}

		}
		return false, 0
	}
	//	fmt.Println("start parrallel")
	var wg sync.WaitGroup
	wg.Add(NCPU)
	for i := 0; i < NCPU; i++ {
		start := i * numpoints / NCPU
		end := (i + 1) * numpoints / NCPU

		go findExistingRoutine(newpoint, t.Points[start:end], start, found, quit, i, &wg)
	}
	wg.Wait()

	var f bool
TestFound:
	select {
	case id = <-found:
		//fmt.Println(id)
		f = true
	default:
		//fmt.Println("found nothing")
		break TestFound
	}

	return f, id

}

func findExistingRoutine(newpoint geo.Point, points []geo.Point, offset int, found chan int, quit chan bool, goroutineid int, wg *sync.WaitGroup) {

	defer wg.Done()
	//	defer fmt.Println("Goroutine exit: ", goroutineid)
	select {
	case <-quit:
		//	fmt.Println("goroutine quit ", goroutineid)
		return
	default:
		for existingpointid, existingpoint := range points {
			if newpoint.Lat() == existingpoint.Lat() && newpoint.Lng() == existingpoint.Lng() {
				found <- existingpointid + offset
				//	fmt.Println("goroutine done found : ", existingpointid+offset)
				quit <- true
				return
			}
		}
		//	fmt.Println("goroutine found nothing: ", goroutineid)
		return
	}
}

func (t *Tracker) generatePointsAndSegments() {
	var frompointid int
	var topointid int
	var pointcount int
	for pipeid, pipe := range t.Pipes {
		/*fmt.Println("len(pipe.Points): ", len(pipe.Points))
		fmt.Println("pipe.Numparts ", pipe.NumPoints) */

		//for each pipe i.e PolyLine
		found := false
		var existingpointid int
		for n, point := range pipe.Points {
			//	start := time.Now()

			//fmt.Println("n: ", n)
			//create new geo.Point
			geopoint := geo.NewPoint(point.Y, point.X)

			found, existingpointid = t.findExisting(*geopoint)

			if !found {
				t.Points = append(t.Points, *geopoint)
				topointid = len(t.Points) - 1
			} else {
				topointid = existingpointid
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
				seg.pipeid = pipeid
				t.Segments = append(t.Segments, *seg)
				//	fmt.Println(seg)
			}

			frompointid = topointid
			pointcount = pointcount + 1
			//	end := time.Since(start)
			//fmt.Println(end)
		}

	}
	t.PointCount = len(t.Points)
	t.SegmentCount = len(t.Segments)
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

	/*for n, con := range t.Connections {
		fmt.Println(con)
		if n > 10 {
			break
		}
	}*/
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

// PathToPoints convert path denoted for id numbers to geo.Point's
func (t Tracker) PathToPoints(path []int) (points []geo.Point) {
	for _, p := range path {
		points = append(points, t.Points[p])
	}
	return points
}

// FindPathID returns point id of path nearest to input point
func (t Tracker) FindPathID(point int) (path []int, found bool) {
	found = false
	path = append(path, point)
	for len(t.Connections[point]) > 0 {
		path = append(path, t.Connections[point][0])
		point = t.Connections[point][0]
		found = true
	}
	return path, found
}

// FindNearestPoint finds the nearest point given a mimimum radius.
// input *geo.Point.
// minimum float64: exclude points closer than this distance.
// returns id, *geo.Point, distance to point.
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
