package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	wr "github.com/mroth/weightedrand"
	"github.com/varlyapp/varlyapp/backend/lib"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type VariantFileType string

const (
	VariantFileTypePng  VariantFileType = "PNG"
	VariantFileTypeTiff                 = "TIFF"
	VariantFileTypeGif                  = "GIF"
)

const FileTypeExpression = `.(png|jpg|jpeg)$`
const RarityExpression = `(.*)#([.0-9]+).(png|jpg|jpeg)$`
const DefaultRarity float64 = 100.0

type CollectionService struct {
	Ctx        context.Context
	Collection *Collection
}

type Trait struct {
	Name      string `json:"name"`
	Collapsed bool   `json:"collapsed"`
}

type Variant struct {
	Name   string  `json:"name"`
	Path   string  `json:"path"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Weight float64 `json:"weight"`
}

type Collection struct {
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	Artist          string               `json:"artist"`
	BaseUri         string               `json:"baseUri"`
	SourceDirectory string               `json:"sourceDirectory"`
	OutputDirectory string               `json:"outputDirectory"`
	Traits          []Trait              `json:"traits"`
	Layers          map[string][]Variant `json:"layers"`
	LayersDummy     []Variant            `json:"layersDummy"`
	Width           float64              `json:"width"`
	Height          float64              `json:"height"`
	Size            int                  `json:"size"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func NewCollectionService() *CollectionService {
	return &CollectionService{}
}

func GetVariantDefaults() *Variant {
	return &Variant{
		Name:   "",
		Path:   "",
		Width:  0.0,
		Height: 0.0,
		// FileType:   VariantFileTypePng,
	}
}

func GetCollectionDefaults() *Collection {
	return &Collection{
		SourceDirectory: "",
		OutputDirectory: "",
		Traits:          []Trait{},
		Layers:          map[string][]Variant{},
		Width:           0.0,
		Height:          0.0,
		Size:            100,
	}
}

func (c *CollectionService) LoadCollection() *Collection {
	collection := &Collection{}
	content, err := lib.OpenFileContents(c.Ctx)

	if err != nil {
		lib.ErrorModal(c.Ctx, "Collection cannot be loaded", "The collection file may be corrupted or data may be missing")
		return nil
	} else {
		err = json.Unmarshal([]byte(content), collection)

		if err != nil {
			lib.ErrorModal(c.Ctx, "Collection cannot be read", err.Error())
			return nil
		}

		return collection
	}
}

// ReadsDirectory reads a direcotory into a Collection
// Items within the directory are split into Layers
func (c *CollectionService) LoadCollectionFromDirectory(dir string) *Collection {
	collection := &Collection{Layers: map[string][]Variant{}}
	lastDirectory := ""

	fileX := regexp.MustCompile(FileTypeExpression)
	rarityX := regexp.MustCompile(RarityExpression)

	err := filepath.Walk(
		dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if dir != path {
					lastDirectory = info.Name()
				}
			} else {
				if fileX.MatchString(info.Name()) {
					var weight float64

					if rarityX.MatchString(path) {
						weight, _ = strconv.ParseFloat(rarityX.ReplaceAllString(path, "$2"), 64)
					}

					if weight <= 0.0 {
						weight = DefaultRarity
					}

					collection.Layers[lastDirectory] = append(collection.Layers[lastDirectory], Variant{Name: info.Name(), Path: path, Weight: weight})
				}
			}

			return nil
		})

	if err != nil {
		lib.ErrorModal(c.Ctx, "Collection cannot be loaded from directory", err.Error())
		return nil
	}

	return collection
}

func (c *CollectionService) SaveCollection(collection *Collection) string {
	file, err := runtime.SaveFileDialog(c.Ctx, runtime.SaveDialogOptions{
		Title:           "Save Varly collection as a file",
		DefaultFilename: collection.Name,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Varly Collection Files (*.json, *.varly)",
				Pattern:     "*.json;*.varly",
			},
		},
	})
	fmt.Println(file)
	if err != nil {
		return ""
	}
	if file == "" {
		lib.ErrorModal(c.Ctx, "Collection has no save location", "Collection destination may be read-only")

		return ""
	}

	contents, err := json.MarshalIndent(collection, "", "  ")

	if err != nil {
		lib.ErrorModal(c.Ctx, "Collection could not be formatted", "Collection data may be in a different format")
		return ""
	}

	err = lib.WriteFileContents(file, contents)

	if err != nil {
		lib.ErrorModal(c.Ctx, "Collection could not be saved", "Collection data may be corrupted")
		return ""
	}

	return file
}

func (c *CollectionService) ValidateCollection() error {
	return nil
}

func (c *CollectionService) GenerateCollection(collection Collection) {
	runtime.EventsEmit(c.Ctx, "collection.generation.started", map[string]int{"CollectionSize": collection.Size})

	var jobs []lib.Job

	for i := 0; i < collection.Size; i++ {
		jobs = append(jobs, lib.Job{Id: i, Config: collection})
	}

	var (
		wg        sync.WaitGroup
		usedDNA   = sync.Map{}
		completed = 0
	)

	wg.Add(1)

	go func() {
		defer wg.Done()
		lib.Batch(c.Ctx, 0, jobs, func(ctx context.Context, id int, job lib.Job) {
			var images []string
			collection := job.Config.(Collection)

			for _, trait := range collection.Traits {
				variants := collection.Layers[trait.Name]

				if len(variants) > 0 {
					var choices []wr.Choice

					for _, variant := range variants {
						choices = append(choices, wr.Choice{Item: variant.Path, Weight: uint(variant.Weight)})
					}

					chooser, err := wr.NewChooser(choices...)

					if err != nil {
						log.Fatal(err)
					}

					pick := chooser.Pick().(string)

					images = append(images, pick)
				}
			}

			defer func() {
				completed++
				data := map[string]string{"ItemNumber": fmt.Sprint(completed), "CollectionSize": fmt.Sprint(collection.Size)}
				runtime.EventsEmit(ctx, "collection.item.generated", data)
			}()

			pngFilepath := fmt.Sprintf("%s/%d.png", collection.OutputDirectory, job.Id)

			dna := lib.GenerateDNA(images)
			val, exists := usedDNA.Load(dna)
			if exists {
				pngFilepath = fmt.Sprintf("%s/duplicate-%d.png", collection.OutputDirectory, job.Id)

				fmt.Println("DNA already exists: ", val)
			} else {
				usedDNA.Store(dna, dna)
			}

			runtime.EventsEmit(ctx, "debug", map[string]interface{}{
				"images": images, "png": pngFilepath,
			})

			err1 := lib.GenerateMetadata(lib.MetadataConfig{
				CollectionName:        collection.Name,
				CollectionSymbol:      "{{SYMBOL}}",
				CollectionDescription: "",
				CollectionBaseURI:     collection.BaseUri,
				Name:                  collection.Name,
				Edition:               job.Id,
				Layers:                images,
				Image:                 pngFilepath,
				Artist:                "Selvin Ortiz",
				DNA:                   dna,
			})

			if err1 != nil {
				fmt.Printf("unable to generate metadata: %s\n%v\n", pngFilepath, err1)
			}

			err2 := lib.GeneratePNG(images, pngFilepath, int(collection.Width), int(collection.Height))

			if err2 != nil {
				fmt.Printf("unable to generate image: %s\n%v\n", pngFilepath, err2)
			}
		})
	}()

	wg.Wait()
}

func (c *CollectionService) GenerateCollectionPreview(collection Collection) string {
	var layers []string

	for _, trait := range collection.Traits {
		variants := collection.Layers[trait.Name]

		if len(variants) > 0 {
			var choices []wr.Choice

			for _, variant := range variants {
				choices = append(choices, wr.Choice{Item: variant.Path, Weight: uint(variant.Weight)})
			}

			chooser, err := wr.NewChooser(choices...)

			if err != nil {
				log.Fatal(err)
			}

			pick := chooser.Pick().(string)
			layers = append(layers, pick)
		}
	}

	preview, err := lib.MakePreview(layers, int(collection.Width), int(collection.Height), 512)

	if err != nil {
		fmt.Println(err)
		lib.ErrorModal(c.Ctx, "No Preview", "Could not generate preview")
	}
	return preview
}
