package main

import (
	"fmt"
	"github.com/jonas-p/go-shp"
	"github.com/kellydunn/golang-geo"
	gj "github.com/kpawlik/geojson"
	"io/ioutil"
	//	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	//"time"
)

// Tracker Struct
type Tracker struct {
	Pipes       []shp.PolyLine
	Points      []geo.Point
	Segments    []Segment
	Connections [][]int
	PipeCount   int
	JoinRadius  int
}

// Segment is a connection between 2 points
type Segment struct {
	Start   *geo.Point
	Startid int
	End     *geo.Point
	Endid   int
	Pipeid  int
}

// OpenFolder open folder of shapefiles
func (t *Tracker) OpenFolder(folder string, radius int) (err error) {
	t.JoinRadius = radius
	if len(folder) < 1 {
		return fmt.Errorf("No folder specified")
	}
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return fmt.Errorf("Opening Folder Failed:, %s ", folder)
	}
	fmt.Println("Opened Folder: folder", folder)
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".shp" {
			path := filepath.Join(folder, file.Name())
			fmt.Println("reading: ", file.Name())
			t.Open(path)

		}
	}

	return err
}

//Open Shape File
func (t *Tracker) Open(file string) {
	ps := len(t.Points)
	ss := len(t.Segments)

	fmt.Println("Reading File", file)

	shape, err := shp.Open(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	for shape.Next() {
		_, p := shape.Shape()

		pl, _ := p.(*shp.PolyLine)
		t.Pipes = append(t.Pipes, *pl)

	}
	shape.Close()
	fmt.Println("Closed File")

	t.PipeCount = len(t.Pipes)
	fmt.Println("Generating Points and Segments")
	t.generatePointsAndSegments()
	fmt.Println("Points: ", len(t.Points)-ps)
	fmt.Println("Segments: ", len(t.Segments)-ss)
	fmt.Println("Creating Connection List")
	t.createConnectionList()

	/*for _, c := range t.Connections {
		if len(c) > 1 {
			fmt.Println(c)
		}
	}*/
}

// findExisting, given a point try to find existing points concurrently
func (t *Tracker) findExisting(newpoint geo.Point, radius int) (exists bool, id int) {
	numpoints := len(t.Points)

	var NCPU int
	NCPU = runtime.NumCPU()

	found := make(chan int, 4)
	quit := make(chan bool, 1)
	var wg sync.WaitGroup

	if numpoints < 100 {
		wg.Add(1)
		go findExistingRoutine(newpoint, t.Points, radius, 0, found, quit, 0, &wg)
	} else {
		for i := 0; i < NCPU; i++ {
			start := i * numpoints / NCPU
			end := (i + 1) * numpoints / NCPU
			wg.Add(1)
			go findExistingRoutine(newpoint, t.Points[start:end], radius, start, found, quit, i, &wg)
		}
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

func findExistingRoutine(newpoint geo.Point, points []geo.Point, radius int, offset int, found chan int, quit chan bool, goroutineid int, wg *sync.WaitGroup) {

	defer wg.Done()
	if radius == 0 {
		for existingpointid, existingpoint := range points {
			select {
			case <-quit:
				return
			default:
				if newpoint.Lat == existingpoint.Lat && newpoint.Lng == existingpoint.Lng {
					found <- existingpointid + offset
					//	fmt.Println("goroutine done found : ", existingpointid+offset)
					quit <- true
					return
				}
			}
		}
	} else {
		for existingpointid, existingpoint := range points {
			select {
			case <-quit:
				return
			default:
				if int(newpoint.GreatCircleDistance(&existingpoint)*1000) < radius {
					found <- existingpointid + offset
					//	fmt.Println("goroutine done found : ", existingpointid+offset)
					quit <- true
					return
				}
			}
		}
	}

}

func (t *Tracker) generatePointsAndSegments() {
	start := time.Now()
	var frompointid int
	var topointid int
	for pipeid, pipe := range t.Pipes {

		//for each pipe i.e PolyLine
		found := false
		var existingpointid int
		end := pipe.NumPoints - 1
		for n, point := range pipe.Points {

			//create new geo.Point
			geopoint := geo.NewPoint(point.Y, point.X)

			if n == 0 || n == int(end) {
				found, existingpointid = t.findExisting(*geopoint, 0)
			} else {
				found, existingpointid = t.findExisting(*geopoint, 0)
			}
			if !found {
				t.Points = append(t.Points, *geopoint)
				topointid = len(t.Points) - 1
			} else {
				topointid = existingpointid
			}

			if n != 0 {
				seg := new(Segment)
				seg.Start = &t.Points[frompointid]
				seg.Startid = frompointid
				seg.End = &t.Points[topointid]
				seg.Endid = topointid
				seg.Pipeid = pipeid
				t.Segments = append(t.Segments, *seg)
			}

			frompointid = topointid

		}

	}

	fmt.Println("Generation took: ", time.Since(start))
}

func (t *Tracker) createConnectionList() {
	start := time.Now()

	var count int
	t.Connections = make([][]int, len(t.Points))
	// fmt.Println(len(t.Connections))
	//fmt.Println(t.SegmentCount)

	for _, segment := range t.Segments {
		from := segment.Startid
		to := segment.Endid

		t.Connections[from] = append(t.Connections[from], to)
		count = count + 1
	}
	fmt.Println("ConnectionCount: ", len(t.Connections))

	fmt.Println("Connection creation took: ", time.Since(start))
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
func (t Tracker) IDsToPoints(path []int) (points []geo.Point) {
	for _, p := range path {
		points = append(points, t.Points[p])
	}
	return points
}

// FindPathID returns point id of path nearest to input point
func (t Tracker) FindPathID(point int) (morepath []int, found bool) {
	found = false
	morepath = append(morepath, point)
	for len(t.Connections[point]) > 0 {
		morepath = append(morepath, t.Connections[point][0])
		point = t.Connections[point][0]
		found = true
	}

	return
}

func (t Tracker) FindPathIDRecursive(point int, path []int, recurse int, joins int) (biggerpath []int, found bool) {
	if recurse == 5 {
		fmt.Println("recursion limit")
		return path, true
	}
	var newpath []int
	var newfound bool
	var fn bool
	if len(path) > 0 {
		end := path[len(path)-1]
		fmt.Println("finding nearest point to end")
		ids := findAllPointsWithin(&t.Points[end], t.Points, float64(t.JoinRadius)/1000.0)
		fmt.Println("path", path)
		fmt.Println("findAllPointsWithin", ids)
		ids = findPointsIDSNotIn(ids, path)
		fmt.Println("subtraction", ids)
		end, _, _, fn = findNearestPointWithin(&t.Points[end], t.IDsToPoints(ids), 0, float64(t.JoinRadius)/1000.0)
		if fn {
			newpath, _ = t.FindPathID(end)
		} else {
			return path, true
		}
	} else {
		newpath, newfound = t.FindPathID(point)
		if !newfound {
			return path, false
		}
	}

	biggerpath = path
	found = true
	for _, id1 := range newpath {
		for _, id2 := range path {
			if id1 == id2 {
				fmt.Println("Recursion: ", recurse)
				fmt.Println("Path loop")
				fmt.Println("joins: ", joins)
				return
			}
		}
	}
	biggerpath = append(path, newpath...)
	return t.FindPathIDRecursive(point, biggerpath, recurse+1, joins+1)
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

func findNearestPointWithin(input *geo.Point, points []geo.Point, minimum float64, maximum float64) (nearestpointid int, nearestpoint *geo.Point, distance float64, found bool) {
	for id, point := range points {
		distance = input.GreatCircleDistance(&point)
		if distance <= maximum && distance > minimum {
			maximum = distance
			nearestpoint = &point
			nearestpointid = id
			found = true
		}

	}
	//fmt.Println("nearestpointid: ", nearestpointid)
	//fmt.Println("nearestpoint:, ", nearestpoint)
	return nearestpointid, nearestpoint, distance, found
}

func findAllPointsWithin(input *geo.Point, points []geo.Point, maximum float64) (ids []int) {
	for id, point := range points {
		distance := input.GreatCircleDistance(&point)
		if distance <= maximum {
			ids = append(ids, id)
		}
	}
	return ids
}

func findPointsIDSNotIn(set []int, subtract []int) (ids []int) {
	// for each point in the set
	for p1 := range set {
		var found bool
		// detect if point exists in the subtraction set
		for p2 := range subtract {
			if p1 == p2 {
				found = true
			}
		}
		// if not found add to output set
		if !found {
			ids = append(ids, p1)
		}
	}
	return ids
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
		coord[0] = gj.Coord(p.Lng)
		coord[1] = gj.Coord(p.Lat)
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
