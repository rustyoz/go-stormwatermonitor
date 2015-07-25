package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/jonas-p/go-shp"
	gj "github.com/rustyoz/geojson"
	"github.com/rustyoz/golang-geo"
)

// OpenShape File
func (t *Tracker) OpenShape(file string) error {
	fmt.Println("GOMAXPROCS", maxproc)
	fmt.Println("Reading File", file)

	shape, err := shp.Open(file)
	if err != nil {
		return err
	}

	for shape.Next() {
		id, p := shape.Shape()

		pl, _ := p.(*shp.PolyLine)
		t.addPipe(polyLineToPipe(pl, id))
	}
	shape.Close()
	fmt.Println("Closed File")

	t.createConnectionList()

	return nil
}

func polyLineToPipe(pl *shp.PolyLine, id int) (pipe Pipe) {
	pipe.ID = id
	pipe.Points = make([]geo.Point, len(pl.Points))
	for n, p := range pl.Points {
		pipe.Points[n] = *geo.NewPoint(p.Y, p.X)
	}
	return pipe
}

func linestringToPipe(ls gj.LineString, id int) (pipe Pipe) {
	pipe.ID = id
	pipe.Points = make([]geo.Point, len(ls.Coordinates))
	for n, p := range ls.Coordinates {

		pipe.Points[n] = *geo.NewPoint(float64(p[1]), float64(p[0]))

	}
	//fmt.Println(pipe)
	return pipe
}

// OpenGeoJSON read GeoJSON file
func (t *Tracker) OpenGeoJSON(filename string) error {
	fmt.Println("GOMAXPROCS", maxproc)
	start := time.Now()
	fmt.Println("Opening: ", filename)
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("OpenJSON() %s", err.Error())
	}

	var data []byte
	data, _ = ioutil.ReadAll(file)
	file.Close()

	var fc gj.FeatureCollection
	fc, err = gj.UnMarshal(data)
	if err != nil {
		return fmt.Errorf("OpenJSON %s", err.Error())
	}
	//fmt.Println(fc)

	for id, f := range fc.Features {
		g, _ := f.GetGeometry()
		t.addPipe(linestringToPipe(*g.(*gj.LineString), id))

	}
	fmt.Println(time.Since(start))
	t.createConnectionList()

	return nil
}
