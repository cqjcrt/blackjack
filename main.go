package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

type Player struct {
	Name     string
	bankroll int
}

type Dealer struct {
	bankroll int
	hand     []Card
}

type Spot struct {
	hand   []Card
	wager  int
	player *Player
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
	players []Player
	shoe    Deck
	discard []Card
	dealer  Dealer
	spots   *[]Spot
	status  GameStatus
	turn    GameTurn
}

type Suit int

const (
	Spade   Suit = 0
	Heart   Suit = 1
	Club    Suit = 2
	Diamond Suit = 3
)

type SpotStatus int

const (
	Ready SpotStatus = iota
	Stand
	Busted
)

type Card struct {
	rank int
	suit Suit
}

type Deck struct {
	cards []Card
}

func main() {
	g := &Game{}
	g.init()
	g.spots[0].player = &Player{Name: "jon"}
	g.deal()
	fmt.Println(g.shoe)
	fmt.Println(g)
	fmt.Println("*****")

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func hello(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	d := buildDeck(1)
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", d[0])
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/hello")
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

func (d *Deck) init() {
	d.cards = make([]Card, 52)
	i := 0
	for r := 1; r < 14; r++ {
		for _, s := range []Suit{Spade, Heart, Club, Diamond} {
			d.cards[i] = Card{r, s}
			i++
		}
	}
	fmt.Println(d.cards)
}

func (d *Deck) shuffle() {
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

func (g *Game) init() {
	// create and shuffle deck
	// TODO: handle creation of more than 1 deck
	d := &Deck{}
	d.init()
	d.shuffle()
	g.shoe = *d

	spots := make([]Spot, 6)
	g.spots = &spots
}

func (g *Game) deal() {
	// check state of game
	// debit wager from player cash
	// deal 1 card to each occupied spot
	for _, sp := range *g.spots {
		fmt.Printf("sp is %v\n", sp)
		if sp.player != nil {
			fmt.Printf("before: %s", sp.hand)
			sp.hand = append(sp.hand, g.shoe.draw())
			fmt.Printf("after: %s", sp.hand)
		}
	}
	// deal to dealer
	fmt.Println("daeler")
	fmt.Println(g.dealer)
	fmt.Println(g.dealer.hand)
	fmt.Println(g.shoe)
	g.dealer.hand = append(g.dealer.hand, g.shoe.draw())

	// deal 1 card to each occupied spot
	for _, sp := range *g.spots {
		fmt.Println("sp is %s", sp)
		if sp.player != nil {
			fmt.Printf("before: %s", sp.hand)
			sp.hand = append(sp.hand, g.shoe.draw())
			fmt.Printf("after: %s", sp.hand)
		}
	}
	// deal to dealer
	fmt.Println("daeler")
	fmt.Println(g.dealer)
	fmt.Println(g.dealer.hand)
	fmt.Println(g.shoe)
	g.dealer.hand = append(g.dealer.hand, g.shoe.draw())

	// set status to PlayerTurn
	// check for dealer blackjack
	// TODO: insurance
	// check for player blackjack
}

func (g *Game) String() string {
	var s strings.Builder
	s.WriteString(fmt.Sprintf("Deck: %s\n", g.shoe))
	s.WriteString(fmt.Sprintf("Dealer: %s\n", g.dealer.hand))
	for _, sp := range *g.spots {
		if sp.player != nil {
			s.WriteString(fmt.Sprintf("Player %s: %s\n", sp.player.Name, sp.hand))
		}
	}
	return s.String()
}

func (g *Game) settle() {
	// compare each spot against dealer hand
	// if dealer win, credit dealer spot's wager
	// if dealer lose, credit spot's player with wager
}

func (g *Game) settleBlackjack() {
	// payout player 3 to 2
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

func buildDeck(numDecks int) []Card {
	deck := make([]Card, 52)
	i := 0
	for r := 1; r < 14; r++ {
		for _, s := range []Suit{Spade, Heart, Club, Diamond} {
			deck[i] = Card{r, s}
			i++
		}
	}
	return deck
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;
    var print = function(message) {
        var d = document.createElement("div");
        d.innerHTML = message;
        output.appendChild(d);
    };
    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("RESPONSE: " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };
    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        print("SEND: " + input.value);
        ws.send(input.value);
        return false;
    };
    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output"></div>
</td></tr></table>
</body>
</html>
`))
