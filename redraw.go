package main

import (
	"fmt"
	"image"
	"os"
	"image/color"
	"math/rand"
	"image/png"
	"time"
	"flag"
	_ "image/jpeg"

)

type rgba struct {
	r, g, b, a uint32
}

type point struct {
	x, y int
}

var ch1 = make(chan rgba)
var ch2 = make(chan rgba)
var chs = make(chan rgba)

func feedChannels(c1, c2, cs rgba) {
	go func() {
		ch1 <- c1
		ch2 <- c2
		chs <- cs
	}()

}

func generateColors(img1 image.Image, img2 image.Image, src *image.Image, line []point) {
	go func() {
		for i := range line {
			r1, g1, b1, a1 := img1.At(line[i].x, line[i].y).RGBA()
			r2, g2, b2, a2 := img2.At(line[i].x, line[i].y).RGBA()
			rs, gs, bs, as := (*src).At(line[i].x, line[i].y).RGBA()

			c1 := rgba{r1, g1, b1, a1}
			//c1 := (r1 - rs + g1 - gs + b1 - bs + a1 - as)
			c2 := rgba{r2, g2, b2, a2}
			//c2 := (r2 - rs + g2 - gs + b2 - bs + a2 - as)
		    cs := rgba{rs, gs, bs, as}

			feedChannels(c1, c2, cs)
			go calcPixelDiff()
			
		}
	}()
}

var d1 = make(chan int)
var d2 = make(chan int)

func calcPixelDiff() {
	
	i1 := <- ch1
	i2 := <- ch2
	cs := <- chs
	//We use sum of difference squared to calculate difference (almost as performant as manhattan distance for better quality)
	c1 := int((i1.r - cs.r)*(i1.r - cs.r) + (i1.g - cs.g)*(i1.g - cs.g) + (i1.b - cs.b)*(i1.b - cs.b) + (i1.a - cs.a)*(i1.a - cs.a))
	c2 := int((i2.r - cs.r)*(i2.r - cs.r) + (i2.g - cs.g)*(i2.g - cs.g) + (i2.b - cs.b)*(i2.b - cs.b) + (i2.a - cs.a)*(i2.a - cs.a))
	
	//once calculations are done, feed them to channels
	d1 <- c1
	d2 <- c2
}

func weighImages(line []point) int{
	//read in the two pictures differences, then whichever has more accurate pixel count is the one saved
	var weightOne int
	var weightTwo int

	//iterate through every pixel changed
	for i := 0; i < len(line); i++ {
		//read in pixel from channels (guaranteed to be same pixel)
		a := <- d1
		b := <- d2
		if a < b {
			weightOne++
		} else {
			weightTwo++
		}
	}
	
	if weightOne > weightTwo {
		return 1
	}

	return 2
}

var pch = make(chan point)
func generatePoints(x0, x1, y0, y1 int) {
	//bresenham line
	//first/eighth octant only
	go func() {	
		dx := x1 - x0
		dy := y1 - y0
		flip := 0
		if dy < 0 {
			flip = 1
			dy = -dy
		}
		
		var err float32 = 0.0	

		deltaerr := float32(dy) / float32(dx)

		y := y0
		for x := x0; x < x1; x++ {
			p := new(point)
			p.x = x;
			p.y = y;

			pch <- *p

			err += deltaerr
			if err >= .5 {
				switch flip {
				case 0:
					y++
				case 1:
					y--
				}
				err -= 1.0
			}
		}
	}()

	
}

func fillLine(line *[]point, dx int) {
	for i := 0; i < dx; i++ {
		//it'd probably be faster to not use a vector but just use a fixed size array
		*line = append(*line, <- pch)
	}
}

func bresenham(line *[]point, x0, x1, y0, y1 int) {
	//spawns child thread to generate points of the line, while the main thread draws them as they're available
	generatePoints(x0, x1, y0, y1)
	fillLine(line, x1 - x0)
}

func openFile() (*os.File, error) {
	reader, err := os.Open(*name)

	return reader, err
}

func decodeImg(file *os.File) (image.Image, error){
	m, _, err := image.Decode(file)

	return m, err
}

func stall() {
	a := 0
	fmt.Printf("Press enter to exit\n")
	fmt.Scanf("%d", &a)
}

func saveImage(name string, img image.Image) error{
	file, err := os.Create(name)
	defer file.Close()
	if err != nil {
		fmt.Printf("Could not save to file: %s\n", name)
		return err
	}
	
	png.Encode(file, img)

	return err
}

var iter = flag.Int("it", 10000, "number of iterations")
var name = flag.String("f", "", "file name.(jpg | png)")
var rad = flag.Int("r", 3, "square side length")

func main() {
	//Parse flags
	flag.Parse()
	startTime := time.Now().Unix()
	rand.Seed(startTime)
	//open the file to be analyzed
	reader, err := openFile()
	defer reader.Close()
	
	if err != nil {
		fmt.Printf("Error: Could not open file\n")
		stall()
		return
	}

	m, err := decodeImg(reader)
	if err != nil {
		fmt.Printf("Error: Could not decode image\n")
		stall()
		return 
	}

	w := m.Bounds().Dx()
	h := m.Bounds().Dy()

	//grab the colors of the image	
	colors := make([]color.Color, 0)
	colorMap := make(map[color.Color]bool)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if _, ok := colorMap[m.At(x, y)]; !ok {
				colorMap[m.At(x,y)] = true
				colors = append(colors, m.At(x,y))
			}
		}
	}

	img1 := image.NewRGBA(m.Bounds())
	img2 := image.NewRGBA(m.Bounds())	

	n := *iter
	//generate
	for i := 0; i < n; i++ {
		//choose random point on image
		x := rand.Intn(w)
		y := rand.Intn(h)

		//choose random box to fill in

		//calculate slope of random line
		lx := (rand.Int() % (*rad-1)) + 1
		ly := (rand.Int() % (*rad-1)) + 1
		yoff := (rand.Int() % (*rad))

		if neg := (rand.Int() % 2); neg == 1 {
			ly = -ly
		}

		line := []point{}
		bresenham(&line, x, lx+x, y+yoff, ly+y)

		//choose random color
		newColor := colors[rand.Intn(len(colors))]

		//set pixels
		for j := range line {
			px := line[j].x
			py := line[j].y
			img1.Set(px, py, newColor)
		}

		//fork off thread to analyze colors
		generateColors(img1, img2, &m, line)

		//compare the two images, copy whichever has more pixels close to the source image (1 is img1, 2 is img2)
		//img2 is used as the "saved progress"
		if weighImages(line) == 1 {
			copy(img2.Pix, img1.Pix)
		} else {
			copy(img1.Pix, img2.Pix)
		}

	}

	saveImage("img.png", img2)

	fmt.Printf("Time Taken: %ds\n", time.Now().Unix() - startTime)
	stall()

}

