package game

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

type Suit string

const (
	Spade   Suit = "s"
	Heart   Suit = "h"
	Club    Suit = "c"
	Diamond Suit = "d"
)

type Card struct {
	Rank string `json:"rank"`
	Suit Suit   `json:"suit"`
}

type Deck struct {
	cards []Card
}

func (s Suit) String() string {
	n := map[Suit]string{
		Spade:   "♠",
		Heart:   "♥",
		Club:    "♣",
		Diamond: "♦",
	}
	return n[s]
}

func (d *Deck) Init() {
	d.cards = make([]Card, 52)
	i := 0
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "T", "J", "Q", "K"}
	for _, r := range ranks {
		for _, s := range []Suit{Spade, Heart, Club, Diamond} {
			d.cards[i] = Card{r, s}
			i++
		}
	}
}

func (d *Deck) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

func (d *Deck) draw() Card {
	var x Card
	if len(d.cards) > 1 {
		x, d.cards = d.cards[0], d.cards[1:]
	} else if len(d.cards) == 1 {
		x, d.cards = d.cards[0], nil
	}
	return x
}

func (d *Deck) String() string {
	var s strings.Builder
	for _, c := range d.cards {
		s.WriteString(c.String())
	}
	s.WriteString(fmt.Sprintf("- Cards left: %d\n", len(d.cards)))
	return s.String()
}

func (c Card) Value() int {
	var value int
	switch c.Rank {
	case "A":
		value = 1
	case "T", "J", "Q", "K":
		value = 10
	default:
		var err error
		value, err = strconv.Atoi(c.Rank)
		if err != nil {
			panic("invalid rank " + c.Rank)
		}
	}
	return value
}

func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Rank, c.Suit)
}
