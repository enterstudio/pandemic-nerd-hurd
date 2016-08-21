package pandemic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const EpidemicsPerGame = 5
const NumInfectionCards = 48

type GameState struct {
	Cities        *Cities        `json:"cities"`
	CityDeck      *CityDeck      `json:"city_deck"`
	DiseaseData   []DiseaseData  `json:"disease_data"`
	InfectionDeck *InfectionDeck `json:"infection_deck"`
	InfectionRate int            `json:"infection_rate"`
	Outbreaks     int            `json:"outbreaks"`
	GameName      string         `json:"game_name"`
}

func NewGame(citiesFile string, gameName string) (*GameState, error) {
	var cities Cities
	data, err := ioutil.ReadFile(citiesFile)
	if err != nil {
		return nil, fmt.Errorf("Could not read cities file at %v: %v", citiesFile, err)
	}
	err = json.Unmarshal(data, &cities)
	if err != nil {
		return nil, fmt.Errorf("Invalid cities JSON file at %v: %v", citiesFile, err)
	}
	cityDeck := CityDeck{}
	cityDeck.All = cities.CityCards(EpidemicsPerGame)

	infectionDeck := NewInfectionDeck(cities.CityNames())
	return &GameState{
		Cities:        &cities,
		DiseaseData:   []DiseaseData{Yellow, Red, Black, Blue, Faded},
		CityDeck:      &cityDeck,
		InfectionDeck: infectionDeck,
		InfectionRate: 2,
		Outbreaks:     0,
		GameName:      gameName,
	}, nil
}

func LoadGame(gameFile string) (*GameState, error) {
	var gameState GameState
	data, err := ioutil.ReadFile(gameFile)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &gameState)
	if err != nil {
		return nil, err
	}
	return &gameState, nil
}

func (gs GameState) Infect(cn CityName) error {
	err := gs.InfectionDeck.Draw(cn)
	if err != nil {
		return err
	}
	city, err := gs.Cities.GetCity(cn)
	if err != nil {
		return err
	}
	// TODO: hanlde if quarantine specialist is present
	if city.Quarantined {
		city.RemoveQuarantine()
		return nil
	}
	// TODO: handle outbreaks
	city.Infect()
	return nil
}

func (gs GameState) Epidemic(cn CityName) error {
	err := gs.InfectionDeck.PullFromBottom(cn)
	if err != nil {
		return err
	}
	err = gs.CityDeck.DrawEpidemic()
	if err != nil {
		return err
	}
	city, _ := gs.Cities.GetCity(cn)
	// TODO: handle if quarantine specialist is present
	if city.Quarantined {
		city.RemoveQuarantine()
		return nil
	}
	// TODO: handle outbreak
	city.Epidemic()
	gs.InfectionDeck.ShuffleDrawn()
	return nil
}

func (gs GameState) Quarantine(cn CityName) error {
	city, err := gs.Cities.GetCity(cn)
	if err != nil {
		return err
	}
	if city.Quarantined {
		return fmt.Errorf("%v is already quarantined", cn)
	}
	city.Quarantine()
	return nil
}

func (gs GameState) RemoveQuarantine(cn CityName) error {
	city, err := gs.Cities.GetCity(cn)
	if err != nil {
		return err
	}
	if !city.Quarantined {
		return fmt.Errorf("%v is not quarantined ", cn)
	}
	city.RemoveQuarantine()
	return nil
}

func (gs GameState) ProbabilityOfCity(cn CityName) float64 {
	city, err := gs.Cities.GetCity(cn)
	if err != nil {
		return 0.0
	}
	if city.Quarantined {
		return 0.0
	}
	// P(epidemic)*P(pull from bottom or from infect drawn) + P(!epidemic)*P(infection deck draw)
	pEpi := gs.CityDeck.probabilityOfEpidemic()
	bottom := gs.InfectionDeck.BottomStriation()
	var pEpiDraw float64
	if bottom.Contains(cn) {
		pEpiDraw = 1.0 / float64(bottom.Size())
	} else if gs.InfectionDeck.Drawn.Contains(cn) {
		pEpiDraw = float64(gs.InfectionRate) / (1.0 + float64(len(gs.InfectionDeck.Drawn)))
	}

	pNoEpiDraw := gs.InfectionDeck.ProbabilityOfDrawing(cn, gs.InfectionRate)
	// fmt.Printf("%v*%v + %v*%v\n", pEpi, pEpiDraw, 1.0-pEpi, pNoEpiDraw)
	return pEpi*pEpiDraw + (1.0-pEpi)*pNoEpiDraw
}

func (gs GameState) CanOutbreak(cn CityName) bool {
	city, err := gs.Cities.GetCity(cn)
	if err != nil {
		return false
	}
	if city.NumInfections == 0 {
		return false
	}
	prob := gs.ProbabilityOfCity(cn)
	if prob == 0.0 {
		return false
	}
	return city.NumInfections == 3 || gs.InfectionDeck.BottomStriation().Contains(cn)
}

func (gs *GameState) GetCity(city CityName) (*City, error) {
	return gs.Cities.GetCity(city)
}

func (gs *GameState) GetDiseaseData(diseaseType DiseaseType) (*DiseaseData, error) {
	for _, data := range gs.DiseaseData {
		if data.Type == diseaseType {
			return &data, nil
		}
	}
	return nil, fmt.Errorf("No disease identified by %v", diseaseType)
}
