package game

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type Suit int

const (
	Spade   Suit = 0
	Heart   Suit = 1
	Club    Suit = 2
	Diamond Suit = 3
)

type Card struct {
	rank int
	suit Suit
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
	for r := 1; r < 14; r++ {
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

func (c Card) String() string {
	var rank string
	switch c.rank {
	case 1:
		rank = "A"
	case 10:
		rank = "T"
	case 11:
		rank = "J"
	case 12:
		rank = "Q"
	case 13:
		rank = "K"
	default:
		rank = fmt.Sprintf("%d", c.rank)
	}

	return fmt.Sprintf("%s%s", rank, c.suit)
}
