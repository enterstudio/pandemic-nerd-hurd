package pandemic

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type CityName string

func (c CityName) String() string {
	return string(c)
}

type CityDeck struct {
	Drawn []CityCard
	All   []CityCard
}

type CityCard struct {
	City       City
	IsEpidemic bool `json:"is_epidemic"`
}

type City struct {
	Name          CityName    `json:"name"`
	Disease       DiseaseType `json:"disease"`
	PanicLevel    PanicLevel  `json:"panic_level"`
	Neighbors     []string    `json:"neighbors"`
	NumInfections int         `json:"num_infections"`
	Quarantined   bool        `json:"quarantined"`
}

type Cities struct {
	Cities []*City `json:"cities"`
}

type byInfectionRate struct {
	names  []CityName
	cities *Cities
}

func (b byInfectionRate) Len() int { return len(b.names) }

func (b byInfectionRate) Swap(i, j int) {
	b.names[i], b.names[j] = b.names[j], b.names[i]
}

func (b byInfectionRate) Less(i, j int) bool {
	nameI := b.names[i]
	nameJ := b.names[j]
	cityI, _ := b.cities.GetCity(nameI)
	cityJ, _ := b.cities.GetCity(nameJ)
	if cityI.NumInfections > cityJ.NumInfections {
		return true
	}
	if cityI.NumInfections < cityJ.NumInfections {
		return false
	}
	return strings.Compare(string(nameI), string(nameJ)) < 0
}

// do we need to model city specializations?
func (c *Cities) CityCards(epidemicCount int) []CityCard {
	cards := []CityCard{}
	for _, city := range c.Cities {
		cards = append(cards, CityCard{*city, false})
	}
	for i := 0; i < epidemicCount; i++ {
		cards = append(cards, CityCard{City{}, true})
	}
	return cards
}

func (c *Cities) SortByInfectionLevel(cities []CityName) []CityName {
	sorted := &byInfectionRate{cities, c}
	sort.Sort(sorted)
	return sorted.names
}

func (c *Cities) GetCityByPrefix(prefix string) (*City, error) {
	var ret *City
	for _, c := range c.Cities {
		c := c
		if strings.HasPrefix(strings.ToLower(string(c.Name)), strings.ToLower(prefix)) {
			if ret != nil {
				return nil, fmt.Errorf("'%v' is ambiguous", prefix)
			}
			ret = c
		}
	}
	if ret == nil {
		return nil, fmt.Errorf("%v is not a prefix for any city", prefix)
	}
	return ret, nil
}

func (c *Cities) GetCity(city CityName) (*City, error) {
	for _, c := range c.Cities {
		if c.Name == CityName(city) {
			return c, nil
		}
	}
	return nil, fmt.Errorf("No city named %v", city)
}

func (c Cities) WithDisease(disease DiseaseType) []*City {
	cities := []*City{}
	for _, city := range c.Cities {
		if city.Disease == disease {
			cities = append(cities, city)
		}
	}
	return cities
}

func (c Cities) CityNames() []CityName {
	names := []CityName{}
	for _, city := range c.Cities {
		names = append(names, city.Name)
	}
	return names
}

func (c *City) Infect() bool {
	if c.NumInfections == 3 {
		return true
	}
	c.NumInfections++
	return false
}

func (c *City) Epidemic() {
	c.NumInfections = 3
}

func (c *City) Quarantine() {
	c.Quarantined = true
}

func (c *City) RemoveQuarantine() {
	c.Quarantined = false
}

func (c *City) SetInfections(infections int) {
	c.NumInfections = infections
}

func (c CityDeck) Total() int {
	return len(c.All)
}

func (c *CityDeck) NumEpidemics() int {
	var totalEpis int
	for _, card := range c.All {
		if card.IsEpidemic {
			totalEpis++
		}
	}
	return totalEpis
}

func (c *CityDeck) cardsPerEpidemic() int {
	// fmt.Printf("cardsPerEpi = %v/%v\n", c.Total(), c.NumEpidemics())
	return c.Total() / c.NumEpidemics()
}

func (c *CityDeck) EpidemicsDrawn() int {
	count := 0
	for _, card := range c.Drawn {
		if card.IsEpidemic {
			count++
		}
	}
	return count
}

func (c *CityDeck) Draw(cn CityName) error {
	for _, card := range c.Drawn {
		if card.City.Name == cn {
			return fmt.Errorf("%v has already been drawn from the city deck", cn)
		}
	}
	for _, card := range c.All {
		if card.City.Name == cn {
			c.Drawn = append(c.Drawn, card)
			return nil
		}
	}
	return fmt.Errorf("No city called %v in the city deck", cn)
}

func (c *CityDeck) DrawEpidemic() error {
	totalEpis := c.NumEpidemics()
	var drawnEpis int
	for _, card := range c.Drawn {
		if card.IsEpidemic {
			drawnEpis++
		}
	}
	if drawnEpis >= totalEpis {
		return fmt.Errorf("Already drawn %v epidemics this game, there shouldn't be any more", drawnEpis)
	}
	c.Drawn = append(c.Drawn, CityCard{City{}, true})
	return nil
}

// 100 city cards, 5 epidemics
// probability of drawing an epidemic on turn 0:
//   1/20 + (1/19 * 19/20)
//
//   1/18 + (1/17 * 17/18)
//
// if an epidemic is drawn, then the probability
// of an epidemic being drawn is 0 until the 10th turn.
//
// if no epidemic is drawn, the probability of drawing
// an epidemic in the 10th turn is 1/2 + (1/1 * 1/2).
//
// on the 11th turn, the probability of the 21st and 22nd cards
// being epidemics is
// 1/20 + (1/19 * 19/20)
//
func (c CityDeck) probabilityOfEpidemic() float64 {
	// fmt.Printf("phase = round(%v/%v)\n", len(c.Drawn), c.cardsPerEpidemic())
	currentPhase := int(math.Floor(float64(len(c.Drawn)) / float64(c.cardsPerEpidemic())))
	// fmt.Printf("%v == %v ?\n", currentPhase, c.EpidemicsDrawn())
	if currentPhase == c.EpidemicsDrawn() {
		return 2.0 / float64(c.cardsPerEpidemic()-(len(c.Drawn)%c.cardsPerEpidemic()))
	} else {
		return 0
	}
}
