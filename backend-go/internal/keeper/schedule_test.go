package keeper

import (
	"context"
	"testing"
	"time"

	"github.com/matchlock/backend-go/internal/cache"
	"github.com/matchlock/backend-go/internal/txline"
)

type fakeScheduleClient struct {
	fixtures []txline.Fixture
	scores   map[int64][]txline.ScoreSnapshotRow
	odds     map[int64][]txline.OddsPayload
}

func (f *fakeScheduleClient) FetchFixturesSnapshot(ctx context.Context, startEpochDay *int) ([]txline.Fixture, error) {
	return f.fixtures, nil
}

func (f *fakeScheduleClient) FetchScoreSnapshot(ctx context.Context, fixtureID int64) ([]txline.ScoreSnapshotRow, error) {
	return f.scores[fixtureID], nil
}

func (f *fakeScheduleClient) FetchOddsSnapshot(ctx context.Context, fixtureID int64) ([]txline.OddsPayload, error) {
	return f.odds[fixtureID], nil
}

func (f *fakeScheduleClient) FetchOddsSnapshotAsOf(ctx context.Context, fixtureID int64, asOf int64) ([]txline.OddsPayload, error) {
	return nil, nil
}

func (f *fakeScheduleClient) FetchOddsUpdates(ctx context.Context, fixtureID int64) ([]txline.OddsPayload, error) {
	return f.odds[fixtureID], nil
}

func TestScheduleWorkerSyncOnce(t *testing.T) {
	cacheStore := newMemCache()
	pastKickoff := time.Now().Add(-2 * time.Hour).UnixMilli()
	worker := &ScheduleWorker{
		Cache: cacheStore,
		Txline: &fakeScheduleClient{
			fixtures: []txline.Fixture{
				{
					FixtureID:          18172379,
					StartTime:          pastKickoff,
					Competition:        "World Cup",
					Participant1:       "USA",
					Participant2:       "Bosnia & Herzegovina",
					Participant1IsHome: true,
				},
			},
			scores: map[int64][]txline.ScoreSnapshotRow{
				18172379: {{
					FixtureIDAlt:       18172379,
					GameStateAlt:       "scheduled",
					SeqAlt:             850,
					Participant1Home:   true,
					Score: &txline.SnapshotScore{
						Participant1: txline.SnapshotTotal{Total: txline.SnapshotGoals{Goals: 2}},
						Participant2: txline.SnapshotTotal{Total: txline.SnapshotGoals{Goals: 0}},
					},
				}},
			},
			odds: map[int64][]txline.OddsPayload{
				18179551: {{
					SuperOddsType: "1X2_PARTICIPANT_RESULT",
					PriceNames:    []string{"part1", "draw", "part2"},
					Prices:        []int32{1330, 5250, 9000},
					Ts:            1,
				}},
			},
		},
	}

	if err := worker.syncOnce(context.Background()); err != nil {
		t.Fatalf("syncOnce: %v", err)
	}
	match, err := cacheStore.GetMatch(context.Background(), "18172379")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if match.Seq != 850 {
		t.Fatalf("seq = %d", match.Seq)
	}
	if match.HomeGoals == nil || *match.HomeGoals != 2 {
		t.Fatalf("home_goals = %#v", match.HomeGoals)
	}
	if match.HomeTeam != "USA" {
		t.Fatalf("home_team = %q", match.HomeTeam)
	}
}

func TestScheduleWorkerPreservesLiveMatch(t *testing.T) {
	cacheStore := newMemCache()
	_ = cacheStore.UpsertMatch(context.Background(), cache.Match{
		MatchID:   "18172379",
		FixtureID: 18172379,
		GameState: "HT",
		Seq:       9,
	})

	worker := &ScheduleWorker{
		Cache: cacheStore,
		Txline: &fakeScheduleClient{fixtures: []txline.Fixture{
			{
				FixtureID:          18172379,
				Competition:        "World Cup",
				Participant1:       "USA",
				Participant2:       "Bosnia & Herzegovina",
				Participant1IsHome: true,
			},
		}},
	}
	if err := worker.syncOnce(context.Background()); err != nil {
		t.Fatalf("syncOnce: %v", err)
	}
	match, err := cacheStore.GetMatch(context.Background(), "18172379")
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if match.GameState != "HT" || match.Seq != 9 {
		t.Fatalf("live state lost: %#v", match)
	}
}

func TestScheduleWorkerRunStopsOnCancel(t *testing.T) {
	worker := &ScheduleWorker{
		Cache:    newMemCache(),
		Txline:   &fakeScheduleClient{},
		Interval: time.Hour,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := worker.Run(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

func TestShouldHydrateScores(t *testing.T) {
	now := time.UnixMilli(1_000_000)
	if shouldHydrateScores(1_500_000, now) {
		t.Fatal("future kickoff should not hydrate")
	}
	if !shouldHydrateScores(1_000_000, now) {
		t.Fatal("kickoff at now should hydrate")
	}
	if !shouldHydrateScores(500_000, now) {
		t.Fatal("past kickoff should hydrate")
	}
}