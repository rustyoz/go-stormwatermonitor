package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	gj "github.com/rustyoz/geojson"
	"github.com/rustyoz/golang-geo"
)

// Tracker Struct
type Tracker struct {
	Points      []geo.Point
	Segments    []Segment
	Connections [][]int
	PipeCount   int
	JoinRadius  int

	PointMap map[geo.Point]int
}

var trackermutex sync.RWMutex

// Segment is a connection between 2 points
type Segment struct {
	Start   *geo.Point
	Startid int
	End     *geo.Point
	Endid   int
	Pipeid  int
}

// Pipe ..
type Pipe struct {
	ID     int
	Points []geo.Point
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
	fmt.Println("Opened Folder: ", folder)
	for _, file := range files {
		if !file.IsDir() {
			path := filepath.Join(folder, file.Name())
			fmt.Println("reading: ", file.Name())
			switch filepath.Ext(file.Name()) {

			case ".shp":
				err = t.OpenShape(path)
			case ".geojson", ".json":
				err = t.OpenGeoJSON(path)

			}
			if err != nil {
				err = fmt.Errorf("OpenFolder() %s", err)
			}
		}

	}
	return err
}

// findExisting, given a point try to find existing points concurrently
func (t *Tracker) findExisting(newpoint geo.Point, radius int) (exists bool, id int) {
	numpoints := len(t.Points)

	found := make(chan int, maxproc)
	quit := make(chan bool, 1)
	var wg sync.WaitGroup

	if numpoints < 100 {
		wg.Add(1)
		go findExistingRoutine(newpoint, t.Points, radius, 0, found, quit, 0, &wg)
	} else {

		for i := 0; i < maxproc; i++ {
			start := i * numpoints / maxproc
			end := (i + 1) * numpoints / maxproc
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

func (t *Tracker) addPipe(pipe Pipe) {
	var frompointid int
	var topointid int

	//for each pipe i.e PolyLine
	found := false
	var existingpointid int
	end := len(pipe.Points) - 1
	for n, point := range pipe.Points {

		if n == 0 || n == int(end) {
			found, existingpointid = t.findExisting(point, 0)
		} else {
			found, existingpointid = t.findExisting(point, 0)
		}
		if !found {
			trackermutex.Lock()
			t.Points = append(t.Points, point)
			trackermutex.Unlock()
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
			seg.Pipeid = pipe.ID
			trackermutex.Lock()
			t.Segments = append(t.Segments, *seg)
			trackermutex.Unlock()
		}

		frompointid = topointid

	}

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

	var ends int
	for _, c := range t.Connections {
		if len(c) == 0 {
			ends = ends + 1
		}
	}
	fmt.Println("Pipe ends: ", ends)

	fmt.Println("ConnectionCount: ", len(t.Connections))

	fmt.Println("Connection creation took: ", time.Since(start))
}

// FindPath return geo.Points of path nearest to input point
func (t *Tracker) FindPath(point int) (path []geo.Point, found bool) {
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
func (t *Tracker) IDsToPoints(path []int) (points []geo.Point) {
	for _, p := range path {
		points = append(points, t.Points[p])
	}
	return points
}

func SelectFromSliceByIndex(indexes []int, slice []int) (result []int) {
	for _, i := range indexes {
		result = append(result, slice[i])
	}
	return result
}

// FindPathID returns point id of path nearest to input point
func (t *Tracker) FindPathID(start int) (morepath []int) {
	fmt.Println("in FindPathID")
	fmt.Println("start = ", start)
	fmt.Println(len(t.Connections[start]))
	for len(t.Connections[start]) > 0 {
		morepath = append(morepath, t.Connections[start][0])
		start = t.Connections[start][0]
	}

	return
}

func (t *Tracker) FindPathIDRecursive(path []int, recurse int, joins int) (biggerpath []int, found bool) {
	fmt.Println("Recursion: ", recurse)
	if recurse == 15 {
		fmt.Println("recursion limit")
		return path, true
	}

	fmt.Println("Start: ", path[0])
	end := path[len(path)-1]
	fmt.Println("End: ", end)

	endgeo := t.Points[end]
	ids := FindAllStartPointsWithin(&endgeo, t.Points, float64(t.JoinRadius)/1000.0, t) // find all start points within the join radius of the end of the path
	fmt.Println("Start points near end\n", ids)
	ids = findPointsIDSNotIn(ids, path)
	fmt.Println("Start points near end not already in the path\n", ids)
	if len(ids) > 0 {
		newstart, _, _, _ := findNearestPointWithin(&endgeo, t.IDsToPoints(ids), 0, float64(t.JoinRadius)/1000.0)
		fmt.Println("Nearest start points not in path: ", ids[newstart])
		joinedpath := append(path, ids[newstart])
		morepath := t.FindPathID(ids[newstart])
		joinedpath = append(joinedpath, morepath...)
		fmt.Println(joinedpath)
		return t.FindPathIDRecursive(joinedpath, recurse+1, joins+1)
	}

	return path, true

}

// FindNearestPoint finds the nearest point given a mimimum radius.
// input *geo.Point.
// minimum float64: exclude points closer than this distance.
// returns id, *geo.Point, distance to point.
func (t *Tracker) FindNearestStartPoint(input *geo.Point, minimum float64) (int, *geo.Point, float64) {
	//fmt.Println("in FindNearestPoint")

	var closestdistance float64
	var nearestpoint *geo.Point
	var nearestpointid int
	closestdistance = 7000000
	for id, point := range t.Points {
		distance := input.GreatCircleDistance(&point)
		if distance < closestdistance && distance > minimum && len(t.Connections[id]) > 0 {
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
		if distance <= maximum && distance > minimum && len(t.Connections[id]) > 0 {
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

func FindAllPointsWithin(input *geo.Point, points []geo.Point, maximum float64) (ids []int) {
	for id, point := range points {
		distance := input.GreatCircleDistance(&point)
		if distance <= maximum {
			ids = append(ids, id)
		}
	}
	return ids
}

func FindAllStartPointsWithin(input *geo.Point, points []geo.Point, maximum float64, t *Tracker) (ids []int) {
	for id, point := range points {
		distance := input.GreatCircleDistance(&point)
		if distance <= maximum && len(t.Connections[id]) > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func findPointsIDSNotIn(set []int, subtract []int) (ids []int) {
	// for each point in the set
	for _, p1 := range set {
		var found bool
		// detect if point exists in the subtraction set
		for _, p2 := range subtract {
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
func (t *Tracker) PathToGeoJSON(path []geo.Point) (string, error) {
	fc := new(gj.FeatureCollection)
	// Start marker
	var p *gj.Feature
	start := new(gj.Point)
	start.Type = `Point`
	c := new(gj.Coordinate)
	c[0] = gj.Coord(path[0].Lng)
	c[1] = gj.Coord(path[0].Lat)
	start.Coordinates = *c

	// Path line
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
	p = gj.NewFeature(start, nil, nil)
	fc.AddFeatures(f, p)

	fc.Type = `FeatureCollection`
	geojson, e := gj.Marshal(fc)
	return geojson, e
}
