package main

import (
    "encoding/json"
    "fmt"
    "image"
    "image/color"
    "image/jpeg"
    _ "image/png"
    "math"
    "os"
    "strconv"
    "sync"
    "sort"
)

type ConfigColors struct {
    Colors []string
}

type ColorPair struct {
    Key color.Color
    Value int
}
type ColorPairList []ColorPair;

func (p ColorPairList) Len() int {return len(p);}
func (p ColorPairList) Less(i int, j int) bool {return p[i].Value < p[j].Value;}
func (p ColorPairList) Swap(i int, j int) {p[i], p[j] = p[j], p[i];}

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
    - c (color.Color): A color.
    - c2 (color.Color): A color.

Returns:
    - uint32: Length of vector.

Example:
    d := GetColorDistance(c, c2);
*/
func GetColorDistance(c color.Color, c2 color.Color) uint32 {
    r, g, b, _ := c.RGBA();
    r2, g2, b2, _ := c2.RGBA();

    dr := float64(AbsDiff(r2, r));
    dg := float64(AbsDiff(g2, g));
    db := float64(AbsDiff(b2, b));

    return uint32(math.Sqrt(math.Pow(dr, 2) + math.Pow(dg, 2) + math.Pow(db, 2)));
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
    var mDistance uint32 = 0xFFFFFFFF;
    index := 0;

    for i := 0; i < len(l); i++ {
        distance := GetColorDistance(c, l[i]);
        if  distance < mDistance {
            mDistance = distance;
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
    defer f.Close();

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

/*
Save a color scheme to a file.

Arguments:
    - p (string): The path to save the color scheme to.
    - c (SaveConfg): The color theme.

Returns:
    - error: An error if any occured.

Example:
    err := SaveConfg("./theme.json", c);
    if err != nil {
        return err;
    }
*/
func SaveConfg(p string, c ConfigColors) error {
    f, err := os.Create(p);
    if err != nil {
        return fmt.Errorf("Error: Failed to create file %s. %w", p, err);
    }
    defer f.Close();

    e := json.NewEncoder(f);
    err = e.Encode(c);
    if err != nil {
        return fmt.Errorf("Error: Failed to write JSON to %s. %w", p, err);
    }

    return nil;
}

/*
Generates a color scheme from an image based on most used colors.

Arguments:
    - p ([][]color.Color): The pixels of the image.
    - m (string): The method to sort by when generating color scheme.

Returns:
    - ConfigColors: The colors for the color theme.

Example:
    c := GenerateColors(p, "min");
*/
func GenerateColors(p [][]color.Color, m string) ConfigColors {
    colorCache := make(map[color.Color]int);
    for i := 0; i < len(p); i++ {
        for j := 0; j < len(p[i]); j++ {
            colorCache[p[i][j]] += 1;
        }
    }

    colorPairs := make(ColorPairList, len(colorCache));
    i := 0;
    for k, v := range colorCache {
        colorPairs[i] = ColorPair{Key: k, Value: v};
        i++; 
    }

    if m == "min" {
        sort.Sort(colorPairs);
    } else {
        sort.Sort(sort.Reverse(colorPairs));
    }

    size := int(math.Min(float64(len(colorPairs)), 21.0))
    colors := make([]string, size);
    for i := 0; i < size; i++ {
        r, g, b, _ := colorPairs[i].Key.RGBA();
        hex := fmt.Sprintf("#%02x%02x%02x", r & 0xFF, g & 0xFF, b & 0xFF);
        colors[i] = hex;
    }

    return ConfigColors{
        Colors: colors,
    };
}

func main() {
    if len(os.Args) < 5 {
        fmt.Fprintf(os.Stderr, "Usage: gowall <convert|generate> <config path> <image path> [save path|min|max]\n");
        return;
    }

    if os.Args[1] == "convert" {
        c, err := LoadConfig(os.Args[2]);
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed to load config. %v\n", err);
            return;
        }

        i, err := LoadImage(os.Args[3]);
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed to load image. %v\n", err);
            return;
        }

        p := LoadPixels(i);
        r := ConvertImage(p, c);
        err = SaveImage(os.Args[4], r);
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed to save image. %v\n", err);
        }
    } else if os.Args[1] == "generate" {
        i, err := LoadImage(os.Args[3]);
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed to load image. %v\n", err);
            return;
        }

        p := LoadPixels(i);
        c := GenerateColors(p, os.Args[4]);
        err = SaveConfg(os.Args[2], c);
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: Failed to save color scheme. %v\n", err);
        }
    }
}
