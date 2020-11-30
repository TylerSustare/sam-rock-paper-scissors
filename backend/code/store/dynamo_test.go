package store

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/tylersustare/sam-rock-paper-scissors/backend/code/game"
)

func TestGameStore(t *testing.T) {
	// use dynamodb local to run the unit tests
	session, err := session.NewSession(&aws.Config{
		Region:   aws.String(os.Getenv("AWS_REGION")),
		Endpoint: aws.String("http://localhost:8000"),
	})
	if err != nil {
		t.Fatalf("unable to create session: %s", err)
	}

	// create test table if it doesn't exist
	testDynamo := dynamodb.New(session)
	_, err = testDynamo.DescribeTable(&dynamodb.DescribeTableInput{TableName: aws.String(os.Getenv("TABLE_NAME"))})
	if err != nil {
		tableName := os.Getenv("TABLE_NAME")

		input := &dynamodb.CreateTableInput{
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("PK"),
					AttributeType: aws.String("S"),
				},
				{
					AttributeName: aws.String("SK"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("PK"),
					KeyType:       aws.String("HASH"),
				},
				{
					AttributeName: aws.String("SK"),
					KeyType:       aws.String("RANGE"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(1),
				WriteCapacityUnits: aws.Int64(1),
			},
			TableName: aws.String(tableName),
		}

		_, err := testDynamo.CreateTable(input)
		if err != nil {
			fmt.Println("Got error calling CreateTable:")
			fmt.Println(err.Error())
			t.Fatalf("unable to store game: %s", err)
		}

		fmt.Println("Created the table", tableName)
	}

	s := New(dynamodb.New(session), os.Getenv("TABLE_NAME"))

	g := game.NewGame()
	// Simulate first player joining
	p1gc, err := game.NewGameContext("first", "1addr", g)

	err = s.StoreAll(g)
	if err != nil {
		t.Fatalf("unable to store game: %s %+v %+v", err, s, g)
	}

	err = s.StorePlayer(p1gc)
	if err != nil {
		t.Fatalf("unable to store player one: %s", err)
	}

	// Simulate second player joining
	p2gc, err := game.NewGameContext("second", "2addr", g)

	err = s.StorePlayer(p2gc)
	if err != nil {
		t.Fatalf("unable to store player two: %s", err)
	}

	// simulate second player reconnecting
	g2, err := s.Load(g.ID)
	if err != nil {
		t.Fatalf("unable to load game from ID: %s", err)
	}
	p2gc2, err := game.NewGameContext("second", "2addr", g2)

	p1gc.Play("rock")

	if p1gc.ActingPlayer.Round != 1 {
		t.Errorf("before store: play was not registered for this round: %+v\n%+v", p1gc.ActingPlayer, p1gc.Game)
	}

	err = s.StorePlay(p1gc)
	if err != nil {
		t.Errorf("couldn't store player 1's play: %s", err)
	}

	if p1gc.Game.Round != 1 {
		t.Errorf("round is not 1: %+v", p1gc.Game)
	}

	if p1gc.ActingPlayer.Round != 1 {
		t.Errorf("after store: play was not registered for this round: %+v\n%+v", p1gc.ActingPlayer, p1gc.Game)
	}

	err = p1gc.Game.AdvanceGame()
	if err == nil {
		t.Errorf("game should not be advanceable with one play")
	}

	if p1gc.Game.PlayCount != 1 {
		t.Errorf("PlayCount should be 1: %+v", p1gc.Game)
	}

	p2gc2.Play("scissors")
	err = s.StorePlay(p2gc2)
	if err != nil {
		t.Errorf("couldn't store player 2's play: %s", err)
	}

	err = p2gc2.Game.AdvanceGame()
	if err != nil {
		t.Errorf("game should be advancable: %s\n%+v", err, p2gc2.Game)
	}

	if p2gc2.ActingPlayer.Score != 0 {
		t.Errorf("player 2 should have no points: %+v", p2gc2.ActingPlayer)
	}

	err = s.StoreRound(p2gc2.Game)
	if err != nil {
		t.Errorf("should be able to advance round: %s\n%+v", err, p2gc2)
	}

	if p2gc2.Game.Round != 2 {
		t.Errorf("round should have advanced: %s\n%+v\n%+v", err, p2gc2.Game, p2gc2.ActingPlayer)
	}
}
