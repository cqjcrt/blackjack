package game

import (
	"fmt"
	"log"
	"strings"
)

type Player struct {
	Name     string
	Bankroll int
}

type Dealer struct {
	bankroll int
	hand     []Card
}

type Spot struct {
	hand   []Card
	Wager  int
	Player *Player
	status SpotStatus
}

type GameStatus int

const (
	Start GameStatus = iota
	PlayerTurn
	End
)

type GameTurn struct {
	spot *Spot
}

type Game struct {
	Players []Player
	shoe    Deck
	discard []Card
	dealer  Dealer
	Spots   []Spot
	status  GameStatus
	turn    GameTurn
}

type SpotStatus int

const (
	Ready SpotStatus = iota
	Stand
	Busted
	Done
)

func (g *Game) Init() {
	// create and shuffle deck
	// TODO: handle creation of more than 1 deck
	d := &Deck{}
	d.Init()
	d.Shuffle()
	g.shoe = *d

	spots := make([]Spot, 6)
	g.Spots = spots
	g.dealer.bankroll = 50000
}

func (g *Game) Next() {
	// this should outline all the states in blackjack
}

func (g *Game) Deal() {
	// check state of game
	// debit wager from player cash
	// deal 1 card to each occupied spot
	for i, sp := range g.Spots {
		if sp.Player != nil {
			c := g.shoe.draw()
			g.Spots[i].hand = append(g.Spots[i].hand, c)
			log.Printf("Player %s draws a %s to hold %s\n", sp.Player.Name, c, g.Spots[i].hand)
		}
	}
	// deal to dealer
	c := g.shoe.draw()
	g.dealer.hand = append(g.dealer.hand, c)
	log.Printf("Dealer draws a %s for a hand of %s\n", c, g.dealer.hand)

	// deal 1 card to each occupied spot
	for i, sp := range g.Spots {
		if sp.Player != nil {
			c := g.shoe.draw()
			g.Spots[i].hand = append(g.Spots[i].hand, c)
			log.Printf("Player %s draws a %s to hold %s\n", sp.Player.Name, c, g.Spots[i].hand)
		}
	}
	// deal to dealer
	c = g.shoe.draw()
	g.dealer.hand = append(g.dealer.hand, c)
	log.Printf("Dealer draws a %s for a hand of %s\n", c, g.dealer.hand)

	g.dealer.hand = []Card{Card{1, 1}, Card{10, 1}}
}

func (g *Game) CheckForDealerBlackjack() bool {
	return len(g.dealer.hand) == 2 && count(g.dealer.hand) == 21
}

func (g *Game) Finish() {
	var dealerDraws bool

	// only draw is there are spots in Stand status
	for _, s := range g.Spots {
		if s.Player != nil && s.status == Stand {
			dealerDraws = true
		}
	}

	if !dealerDraws {
		log.Printf("No active hands remain. Dealer doesn't need to draw")
		return
	}

	// TODO: handle soft 17
	for count(g.dealer.hand) < 17 || (isSoft(g.dealer.hand) && count(g.dealer.hand) == 17) {
		c := g.shoe.draw()
		g.dealer.hand = append(g.dealer.hand, c)

		// check if all spots are won, stand, busted
		if count(g.dealer.hand) > 21 {
			log.Printf("Dealer busts with %s for a hand of %s (%d)\n", c, g.dealer.hand, count(g.dealer.hand))
		} else {
			log.Printf("Dealer draws a %s for a hand of %s (%d)\n", c, g.dealer.hand, count(g.dealer.hand))
		}
	}

}

func (g *Game) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Deck: %s\n", g.shoe))
	s.WriteString(fmt.Sprintf("Dealer: %s\n", g.dealer.hand))
	for _, sp := range g.Spots {
		if sp.Player != nil {
			s.WriteString(fmt.Sprintf("Player %s: %s\n", sp.Player.Name, sp.hand))
		}
	}
	return s.String()
}

func (g *Game) Settle() {
	dealerCount := count(g.dealer.hand)
	// compare each spot against dealer hand
	for i, sp := range g.Spots {
		if sp.Player != nil {
			playerCount := count(sp.hand)
			if sp.status == Stand {
				if dealerCount > 21 || (playerCount <= 21 && playerCount > dealerCount) {
					// player win
					log.Printf("Player wins with %s (%d) against %s (%d)\n", sp.hand, playerCount, g.dealer.hand, dealerCount)
					// credit
					g.playerWins(i, false)
				} else if dealerCount > playerCount && dealerCount <= 21 {
					// player lose
					log.Printf("Player loses with %s (%d) against %s (%d)\n", sp.hand, playerCount, g.dealer.hand, dealerCount)
					// debit
					g.playerLoses(i)
				} else if dealerCount == playerCount && dealerCount <= 21 {
					log.Printf("Player pushes with %s (%d) against %s (%d)\n", sp.hand, playerCount, g.dealer.hand, dealerCount)
				}
			} else if sp.status == Busted {
				// player busted, debit
				g.playerLoses(i)
				log.Printf("Player loses with %s (%d) against %s (%d)\n", sp.hand, playerCount, g.dealer.hand, dealerCount)
			}
		}
	}
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
	g.Spots[i].Player.Bankroll += wager
	g.dealer.bankroll -= wager
	log.Printf("Player %s wins %d. (Bankroll %d)\n", g.Spots[i].Player.Name, wager, g.Spots[i].Player.Bankroll)

}

func (g *Game) playerLoses(i int) {
	// TODO: synchronize?
	g.Spots[i].Player.Bankroll -= g.Spots[i].Wager
	g.dealer.bankroll += g.Spots[i].Wager
	log.Printf("Player %s loses %d. (Bankroll %d)\n", g.Spots[i].Player.Name, g.Spots[i].Wager, g.Spots[i].Player.Bankroll)
}

func (g *Game) Hit(i int) {
	if g.Spots[i].status != Ready {
		log.Printf("Player %s cannot hit with %s (%v)\n", g.Spots[i].Player.Name, g.Spots[i].hand, g.Spots[i].status)
		return
	}

	c := g.shoe.draw()
	g.Spots[i].hand = append(g.Spots[i].hand, c)

	// do evaluate here? check for bust?
	if count(g.Spots[i].hand) > 21 {
		g.Spots[i].status = Busted
		log.Printf("Player %s busts with %s for a hand of %s\n", g.Spots[i].Player.Name, c, g.Spots[i].hand)
		return
	}

	log.Printf("Player %s hits and recieves %s for a hand of %s\n", g.Spots[i].Player.Name, c, g.Spots[i].hand)
}

func (g *Game) Stand(i int) {
	if g.Spots[i].status == Ready {
		g.Spots[i].status = Stand
		log.Printf("Player %s stands with %s\n", g.Spots[i].Player.Name, g.Spots[i].hand)
	} else {
		log.Printf("Player %s cannot stand with %s (%v)\n", g.Spots[i].Player.Name, g.Spots[i].hand, g.Spots[i].status)
	}
}

func count(hand []Card) int {
	// TODO: handle soft 17
	var hasAce bool
	var count int

	for _, c := range hand {
		if c.rank > 10 {
			count += 10
		} else if c.rank == 1 {
			hasAce = true
			count++
		} else if c.rank > 1 && c.rank <= 10 {
			count += c.rank
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
		if c.rank > 10 {
			count += 10
		} else if c.rank == 1 {
			hasAce = true
			count++
		} else if c.rank > 1 && c.rank <= 10 {
			count += c.rank
		}
	}

	// count an ace as "11"
	return hasAce && count < 12
}
