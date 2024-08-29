package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strconv"
	"sync"
)

type ConfigColors struct {
    Colors []string
}

/*
Returns the absolute difference of two numbers.

Arguments:
    - x (uint32): A value.
    - y (uint32): A value.

Returns:
    - uint32: The calculated difference.

Example:
    a := AbsDiff(x, y);
*/
func AbsDiff(x uint32, y uint32) uint32 {
    if x > y {
        return x - y;
    }
    return y - x;
}

/*
Returns the length of vector.

Arguments
    - r, g, b (unint32): The vector component lengths.

Returns:
    - uint32: Length of vector.

Example:
    d := GetDistance(128, 43, 255);
*/
func GetDistance(r uint32, g uint32, b uint32) uint32 {
    r2 := float64(r);
    g2 := float64(g);
    b2 := float64(b);
    return uint32(math.Sqrt(math.Pow(r2, 2) + math.Pow(g2, 2) + math.Pow(b2, 2)));
}

/*
Gets the closest color matching given color against a list of colors.

Arguments:
    - c (color.Color): The color to match.
    - l ([]color.Color): This list of colors to match against.

Returns
    - color.Color: The closest color.

Example:
    c := GetClosestColor(toMatch, colors);
*/
func GetClosestColor(c color.Color, l []color.Color) color.Color {
    var rDistance uint32 = 0xFFFF;
    var gDistance uint32 = 0xFFFF;
    var bDistance uint32 = 0xFFFF;
    index := 0;

    for i := 0; i < len(l); i++ {
        r, g, b, _ := c.RGBA();
        r2, g2, b2, _ := l[i].RGBA();

        rDiff := AbsDiff(r2, r);
        gDiff := AbsDiff(g2, g);
        bDiff := AbsDiff(b2, b);
        
        if GetDistance(rDiff, gDiff, bDiff) < GetDistance(rDistance, gDistance, bDistance) {
            rDistance = rDiff;
            gDistance = gDiff;
            bDistance = bDiff;
            index = i;
        }
    }

    return l[index];
}

/* 
Returns an image.Image based on given filepath.

Arguments:
    - p (string): The filepath to the image.

Returns:
    - image.Image: The image loaded from given filepath.
    - error: The error if any occured trying to load given image.

Example:
    i, e := LoadImage("./Picture.png");
    if e != nil {
        return e;
    }
*/
func LoadImage(p string) (image.Image, error) {
    f, err := os.Open(p);
    if err != nil {
        return nil, fmt.Errorf("Error: Could not open image %s. %w", p, err);
    }

    i, _, err := image.Decode(f);
    if err != nil {
        return nil, fmt.Errorf("Error: Failed to decode image %s. %w", p, err);
    }

    return i, nil;
}

/*
Returns a matrix of pixels from an image.

Arguments:
    - pic (image.Image): The image to get the pixels from.

Returns:
    - [][]color.Color: The matrix of pixels from the image. 

Example:
    p := LoadPixels(i);
*/
func LoadPixels(pic image.Image) [][]color.Color {
    maxX := pic.Bounds().Max.X;
    maxY := pic.Bounds().Max.Y;

    var wg sync.WaitGroup;
    wg.Add(maxY);
    
    result := make([][]color.Color, maxY);
    for i := 0; i < maxY; i++ {
        result[i] = make([]color.Color, maxX);
        go func(row int) {
            defer wg.Done();
            for j := 0; j < maxX; j++ {
                result[row][j] = pic.At(j, row);
            }
        }(i);
    }

    wg.Wait();
    return result;
}

/*
Create a new image with colors matching the given pallete.

Arguments:
    - p ([][]color.Color): The image to be converted.
    - c ([]color.Color): The pallete to convert to.

Returns:
    - image.Image: The converted image.
*/
func ConvertImage(p [][]color.Color, c []color.Color) image.Image {
    result := image.NewNRGBA(
        image.Rectangle{
            Min: image.Point{X: 0, Y: 0},
            Max: image.Point{X: len(p[0]), Y: len(p)},
        },
    );

    var wg sync.WaitGroup;
    wg.Add(len(p));

    var mutex sync.RWMutex;
    colorCache := make(map[color.Color]color.Color);
    for i := 0; i < len(p); i++ {
        go func(row int) {
            defer wg.Done();
            for j := 0; j < len(p[row]); j++ {
                mutex.RLock();
                cachedValue := colorCache[p[row][j]];
                mutex.RUnlock();
                if cachedValue != nil {
                    r, g, b, a := cachedValue.RGBA();
                    result.Pix[(row * result.Stride) + (j * 4)] = uint8(r);
                    result.Pix[(row * result.Stride) + (j * 4) + 1] = uint8(g);
                    result.Pix[row * result.Stride + (j * 4) + 2] = uint8(b);
                    result.Pix[row * result.Stride + (j * 4) + 3] = uint8(a);
                } else {
                    closestColor := GetClosestColor(p[row][j], c);
                    r, g, b, a := closestColor.RGBA();
                    result.Pix[(row * result.Stride) + (j * 4)] = uint8(r);
                    result.Pix[(row * result.Stride) + (j * 4) + 1] = uint8(g);
                    result.Pix[(row * result.Stride) + (j * 4) + 2] = uint8(b);
                    result.Pix[(row * result.Stride) + (j * 4) + 3] = uint8(a);

                    mutex.Lock();
                    if colorCache[p[row][j]] == nil {
                        colorCache[p[row][j]] = closestColor;
                    }
                    mutex.Unlock();
                }
            }
        }(i);
    }

    wg.Wait();
    return result;
}

/*
Saves an image to a path.

Arguments:
    - p (string): Path to save image.
    - i (image.Image): Image to save.

Returns:
    - error: Error saving the image if any.

Example:
    err := SaveImage("./test.jpeg", i);
    if err != nil {
        return err;
    }
*/
func SaveImage(p string, i image.Image) error {
    f, err := os.Create(p);
    if err != nil {
        return fmt.Errorf("Error: Cannot create file %s. %w", p, err);
    }

    err = jpeg.Encode(f, i, nil);
    if err != nil {
        return fmt.Errorf("Error: Failed to encode image %s. %w", p, err);
    }

    return nil;
}

/*
Returns a list of colors from the config file.

Arguments
    - p (string): The file path to the config file.

Returns:
    - []color.Color: The list of loaded colors.
    - error: The error that occured when attempting to load then from the file if any.

Example:
    c, err := LoadConfig("./config.json");
    if err != nil {
        return err;
    }
*/
func LoadConfig(p string) ([]color.Color, error) {
    f, err := os.Open(p);
    if err != nil {
        return nil, fmt.Errorf("Error: Could not open config file %s. %w", p, err);
    }

    var v ConfigColors;
    d := json.NewDecoder(f);
    err = d.Decode(&v);
    if err != nil {
        return nil, fmt.Errorf("Error: Could not parse config file %s. %w", p, err);
    }

    result := make([]color.Color, len(v.Colors));
    for i := 0; i < len(v.Colors); i++ {
        num, err := strconv.ParseUint(v.Colors[i][1:], 16, 32);
        if err != nil {
            return nil, fmt.Errorf("Error: Failed to parse color %s. %w", v.Colors[i], err);
        }

        result[i] = color.NRGBA{
            R: uint8(num >> 16),
            G: uint8((num >> 8) & 0xFF),
            B: uint8(num & 0xFF),
            A: uint8(0xFF),
        };
    }

    return result, nil;
}

func main() {
    if len(os.Args) < 4 {
        fmt.Fprintf(os.Stderr, "Usage ./gowall <config path> <image path> <save path>\n");
        return;
    }

    c, err := LoadConfig(os.Args[1]);
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: Failed to load config. %v\n", err);
        return;
    }

    i, err := LoadImage(os.Args[2]);
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: Failed to load image. %v\n", err);
        return;
    }

    p := LoadPixels(i);
    r := ConvertImage(p, c);
    err = SaveImage(os.Args[3], r);
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: Failed to save image. %v\n", err);
        return;
    }
}
