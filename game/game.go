package game

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

type Player struct {
	Name    string    `json:"name"`
	ID      uuid.UUID `json:"id"`
	Balance int       `json:"balance"`
}

type Hand []Card

type Dealer struct {
	balance  int
	HoleCard Card `json:"holeCard"`
	hand     Hand
}

type Spot struct {
	ID       uuid.UUID  `json:"id"`
	Hand     Hand       `json:"hand"`
	Wager    int        `json:"wager"`
	PlayerID uuid.UUID  `json:"playerId"`
	Status   SpotStatus `json:"status"`
	Next     *Spot      `json:"-"`
}

type SpotStatus int

type BustedError struct {
	Hand Hand
}

func (be *BustedError) Error() string {
	return fmt.Sprintf("you busted: %s", be.Hand)
}

const (
	NoWager SpotStatus = iota
	Wagered
	CanHit
	Stand
	Busted
	Done
)

type GameStatus int

const (
	Wager GameStatus = iota
	Dealing
	PlayerTurn
	DealerTurn
)

type GameTurn struct {
	spot *Spot
}

type Game struct {
	Players map[uuid.UUID]*Player `json:"-"`
	shoe    Deck                  `json:"-"`
	discard []Card                `json:"-"`
	Dealer  Dealer                `json:"dealer"`
	Spots   []Spot                `json:"spots"`
	current *Spot                 `json:"-"`
	Status  GameStatus            `json:"status"`
}

func (h Hand) Peek() Card {
	return ([]Card)(h)[0]
}

func (g *Game) Init() {
	// create and shuffle deck
	// TODO: handle creation of more than 1 deck
	// TODO: add cut card for shuffle
	d := &Deck{}
	d.Init()
	d.Shuffle()
	g.shoe = *d

	spots := make([]Spot, 6)
	for i := 0; i < 5; i++ {
		spots[i].Next = &spots[i+1]
	}
	g.Spots = spots
	g.Players = make(map[uuid.UUID]*Player)
	g.Dealer.balance = 50000
}

func (g *Game) AddPlayer(name string) *Spot {
	id, err := uuid.NewRandom()
	if err != nil {
		panic("could not generate uuid for player")
	}
	p := Player{Name: name, Balance: 1000, ID: id}
	g.Players[id] = &p

	for i, s := range g.Spots {
		if s.PlayerID == uuid.Nil {
			g.Spots[i].PlayerID = p.ID
			g.Spots[i].ID, err = uuid.NewRandom()
			if err != nil {
				log.Printf("game: error creating guid for new spot")
			}
			return &g.Spots[i]
		}
	}
	return nil
}

func (g *Game) PlayerWagered() bool {
	for _, s := range g.Spots {
		if s.Occupied() && s.Status != Wagered {
			log.Println(s.Status)
			return false
		}
		log.Println("skipping")
	}
	return true
}

func (g *Game) PlayerStandOrBusted() bool {
	for _, s := range g.Spots {
		if s.Occupied() && (s.Status != Stand && s.Status != Busted) {
			return false
		}
	}
	return true
}

func (g *Game) Draw() Card {
	return g.shoe.draw()
}

func (g *Game) Next() *Spot {
	// this should outline all the states in blackjack
	g.current = g.current.Next
	return g.current
}

func (g *Game) Cleanup() {
	// moves everything to discard
	for i, s := range g.Spots {
		g.discard = append(g.discard, ([]Card)(s.Hand)...)
		g.Spots[i].Hand = nil
		g.Spots[i].Status = NoWager
	}
	g.discard = append(g.discard, ([]Card)(g.Dealer.hand)...)
	g.Dealer.hand = nil

	g.Status = Wager
}

func (g *Game) Deal() error {
	// check state of game
	// check deck penetration, if we need to shuffle
	if g.Status != Wager {
		log.Printf("Game in progress, cannot deal a new game.")
		return fmt.Errorf("Game in progress, cannot deal a new game.")
	}

	g.Status = Dealing

	// debit wager from player cash
	// deal 1 card to each occupied spot
	for i, sp := range g.Spots {
		if sp.PlayerID != uuid.Nil {
			c := g.shoe.draw()
			g.Spots[i].Hand = append(g.Spots[i].Hand, c)
			log.Printf("Player %s draws a %s to hold %s\n", sp.PlayerID, c, g.Spots[i].Hand)
		}
	}
	// deal to dealer
	c := g.shoe.draw()
	g.Dealer.hand = append(g.Dealer.hand, c)
	g.Dealer.HoleCard = c
	log.Printf("Dealer draws a %s for a hand of %s\n", c, g.Dealer.hand)

	// deal 1 card to each occupied spot
	for i, sp := range g.Spots {
		if sp.PlayerID != uuid.Nil {
			c := g.shoe.draw()
			g.Spots[i].Hand = append(g.Spots[i].Hand, c)
			log.Printf("Player %s draws a %s to hold %s\n", sp.PlayerID, c, g.Spots[i].Hand)
			g.Spots[i].Status = CanHit
		}
	}
	// deal to dealer (second card is hidden)
	c = g.shoe.draw()
	g.Dealer.hand = append(g.Dealer.hand, c)
	log.Printf("Dealer draws a second card and is showing %s\n", g.Dealer.hand.Peek())

	// force bj
	// g.Dealer.hand = []Card{Card{1, 1}, Card{10, 1}}

	// assign current spot to first spot
	g.current = &g.Spots[0]

	g.Status = PlayerTurn

	return nil
}

func (g *Game) CheckForDealerBlackjack() bool {
	return len(g.Dealer.hand) == 2 && count(g.Dealer.hand) == 21
}

func (g *Game) Finish() {
	var dealerDraws bool
	g.Status = DealerTurn

	// only draw is there are spots in Stand status
	for _, s := range g.Spots {
		if s.PlayerID != uuid.Nil && s.Status == Stand {
			dealerDraws = true
		}
	}

	if !dealerDraws {
		log.Printf("No active hands remain. Dealer doesn't need to draw")
		return
	}

	// TODO: handle soft 17
	for count(g.Dealer.hand) < 17 || (isSoft(g.Dealer.hand) && count(g.Dealer.hand) == 17) {
		c := g.shoe.draw()
		g.Dealer.hand = append(g.Dealer.hand, c)

		// check if all spots are won, stand, busted
		if count(g.Dealer.hand) > 21 {
			log.Printf("Dealer busts with %s for a hand of %s (%d)\n", c, g.Dealer.hand, count(g.Dealer.hand))
		} else {
			log.Printf("Dealer draws a %s for a hand of %s (%d)\n", c, g.Dealer.hand, count(g.Dealer.hand))
		}
	}
}

func (g *Game) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Deck: %s\n", g.shoe))
	s.WriteString(fmt.Sprintf("Dealer: %s\n", g.Dealer.hand))
	for _, sp := range g.Spots {
		if sp.PlayerID != uuid.Nil {
			s.WriteString(fmt.Sprintf("Player %s: %s\n", sp.PlayerID, sp.Hand))
		}
	}
	return s.String()
}

func (g *Game) Settle() {
	dealerCount := count(g.Dealer.hand)
	// compare each spot against dealer hand
	for i, sp := range g.Spots {
		if sp.PlayerID != uuid.Nil {
			playerCount := count(sp.Hand)
			if g.Status == DealerTurn || sp.Status == Stand {
				if dealerCount > 21 || (playerCount <= 21 && playerCount > dealerCount) {
					// player win
					log.Printf("Player wins with %s (%d) against %s (%d)\n", sp.Hand, playerCount, g.Dealer.hand, dealerCount)
					// credit
					g.playerWins(i, false)
				} else if dealerCount > playerCount && dealerCount <= 21 {
					// player lose
					log.Printf("Player loses with %s (%d) against %s (%d)\n", sp.Hand, playerCount, g.Dealer.hand, dealerCount)
					// debit
					g.playerLoses(i)
				} else if dealerCount == playerCount && dealerCount <= 21 {
					log.Printf("Player pushes with %s (%d) against %s (%d)\n", sp.Hand, playerCount, g.Dealer.hand, dealerCount)
				}
			} else if sp.Status == Busted {
				// player busted, debit
				g.playerLoses(i)
				log.Printf("Player loses with %s (%d) against %s (%d)\n", sp.Hand, playerCount, g.Dealer.hand, dealerCount)
			}
		}
	}
	g.Status = DealerTurn
}

func (g *Game) settleBlackjack() {
	// payout player 3 to 2
}

func (g *Game) playerWins(i int, blackjack bool) {
	// TODO: synchronize?
	wager := g.Spots[i].Wager
	if blackjack {
		wager *= 2 // TODO: handle floating point 3:2
	}
	p := g.Players[g.Spots[i].PlayerID]
	p.Balance += wager
	g.Dealer.balance -= wager
	log.Printf("Player %s wins %d. (Balance %d)\n", p.Name, wager, p.Balance)

}

func (g *Game) playerLoses(i int) {
	// TODO: synchronize?
	p := g.Players[g.Spots[i].PlayerID]
	p.Balance -= g.Spots[i].Wager
	g.Dealer.balance += g.Spots[i].Wager
	log.Printf("Player %s loses %d. (Balance %d)\n", p.Name, g.Spots[i].Wager, p.Balance)
}

func (g *Game) DealerHand() Hand {
	if g.Status != DealerTurn {
		log.Printf("game: can't reveal dealer hand")
		return nil
	}

	return g.Dealer.hand
}

func (s *Spot) Double(c Card) (*Card, error) {
	// if g.Status != PlayerTurn {
	// 	return nil, errors.New("game: not player's turn to double")
	// }

	// player must have only 2 cards
	if len(s.Hand) != 2 {
		msg := fmt.Sprintf("Player %s has %d cards; therefore cannot double", s.PlayerID, len(s.Hand))
		log.Print(msg)
		return nil, fmt.Errorf(msg)
	}
	if s.Status != CanHit {
		msg := fmt.Sprintf("Player %s cannot hit with %s (%v)\n", s.PlayerID, s.Hand, s.Status)
		log.Print(msg)
		return nil, fmt.Errorf(msg)
	}

	s.Wager *= 2
	s.Hand = append(s.Hand, c)

	// do evaluate here? check for bust?
	if count(s.Hand) > 21 {
		s.Status = Busted
		msg := fmt.Sprintf("Player %s busts with %s for a hand of %s\n", s.PlayerID, c, s.Hand)
		return nil, fmt.Errorf(msg)
	}

	log.Printf("Player %s double and recieves %s for a hand of %s\n", s.PlayerID, c, s.Hand)
	s.Status = Stand
	return &c, nil
}

func (s *Spot) Hit(c Card) (*Card, error) {
	// if g.Status != PlayerTurn {
	// 	return nil, errors.New("game: not player's turn to hit")
	// }

	if s.Status != CanHit {
		msg := fmt.Sprintf("Player %s cannot hit with %s (%v)\n", s.PlayerID, s.Hand, s.Status)
		log.Print(msg)
		return nil, fmt.Errorf(msg)
	}

	s.Hand = append(s.Hand, c)

	// do evaluate here? check for bust?
	if count(s.Hand) > 21 {
		s.Status = Busted
		msg := fmt.Sprintf("Player %s busts with %s for a hand of %s\n", s.PlayerID, c, s.Hand)
		log.Print(msg)
		// set next pointer to next spot
		// g.Next()
		return nil, &BustedError{Hand: s.Hand}
	}

	log.Printf("Player %s hits and recieves %s for a hand of %s\n", s.PlayerID, c, s.Hand)
	return &c, nil
}

func (s *Spot) Stand() error {
	if s.Status == CanHit {
		s.Status = Stand
		log.Printf("Player %s stands with %s\n", s.PlayerID, s.Hand)
	} else {
		log.Printf("Player %s cannot stand with %s (%v)\n", s.PlayerID, s.Hand, s.Status)
	}
	return nil
}

func (s *Spot) Occupied() bool {
	return s.PlayerID != uuid.Nil
}

func count(hand []Card) int {
	// TODO: handle soft 17
	var hasAce bool
	var count int

	for _, c := range hand {
		if c.Value() > 10 {
			count += 10
		} else if c.Value() == 1 {
			hasAce = true
			count++
		} else if c.Value() > 1 && c.Value() <= 10 {
			count += c.Value()
		}
	}

	// count an ace as "11"
	if hasAce && count < 12 {
		count += 10
	}
	return count
}

func isSoft(hand []Card) bool {
	// TODO: handle soft 17
	var hasAce bool
	var count int

	for _, c := range hand {
		if c.Value() > 10 {
			count += 10
		} else if c.Value() == 1 {
			hasAce = true
			count++
		} else if c.Value() > 1 && c.Value() <= 10 {
			count += c.Value()
		}
	}

	// count an ace as "11"
	return hasAce && count < 12
}
