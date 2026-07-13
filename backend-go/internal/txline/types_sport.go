package txline

// Sport-specific nested types for /api/scores/stream (Scores schema).

// SoccerScore is per-period or total goal/card stats for soccer.
type SoccerScore struct {
	Goals       int32 `json:"Goals"`
	YellowCards int32 `json:"YellowCards"`
	RedCards    int32 `json:"RedCards"`
	Corners     int32 `json:"Corners"`
}

// SoccerTotalScore aggregates soccer stats across periods.
type SoccerTotalScore struct {
	Goals int32       `json:"Goals"`
	H1    *SoccerScore `json:"H1,omitempty"`
	HT    *SoccerScore `json:"HT,omitempty"`
	H2    *SoccerScore `json:"H2,omitempty"`
	ET1   *SoccerScore `json:"ET1,omitempty"`
	ET2   *SoccerScore `json:"ET2,omitempty"`
	PE    *SoccerScore `json:"PE,omitempty"`
	ETTotal *SoccerScore `json:"ETTotal,omitempty"`
	Total *SoccerScore `json:"Total,omitempty"`
}

// GoalCount returns the best available goal tally for settlement/display.
func (s SoccerTotalScore) GoalCount() int32 {
	if s.Goals != 0 {
		return s.Goals
	}
	if s.Total != nil {
		return s.Total.Goals
	}
	return 0
}

// SoccerFixtureScore is the live soccer scoreboard payload.
type SoccerFixtureScore struct {
	Participant1 SoccerTotalScore `json:"Participant1"`
	Participant2 SoccerTotalScore `json:"Participant2"`
}

// SoccerFixtureClock is the running match clock for soccer.
type SoccerFixtureClock struct {
	Running bool  `json:"running"`
	Seconds int32 `json:"seconds"`
	RunningAlt bool `json:"Running"`
	SecondsAlt int32 `json:"Seconds"`
}

func (c SoccerFixtureClock) IsRunning() bool {
	return c.Running || c.RunningAlt
}

func (c SoccerFixtureClock) ElapsedSeconds() int32 {
	if c.Seconds != 0 {
		return c.Seconds
	}
	return c.SecondsAlt
}

// SoccerData is an in-play soccer event payload.
type SoccerData struct {
	Action       string `json:"Action,omitempty"`
	Color        string `json:"Color,omitempty"`
	Corner       bool   `json:"Corner,omitempty"`
	FreeKickType string `json:"FreeKickType,omitempty"`
	Goal         bool   `json:"Goal,omitempty"`
	GoalType     string `json:"GoalType,omitempty"`
	Minutes      int32  `json:"Minutes,omitempty"`
	Outcome      string `json:"Outcome,omitempty"`
	Participant  int32  `json:"Participant,omitempty"`
	Penalty      bool   `json:"Penalty,omitempty"`
	PlayerID     int32  `json:"PlayerId,omitempty"`
	PlayerInID   int32  `json:"PlayerInId,omitempty"`
	PlayerOutID  int32  `json:"PlayerOutId,omitempty"`
	RedCard      bool   `json:"RedCard,omitempty"`
	YellowCard   bool   `json:"YellowCard,omitempty"`
	VAR          bool   `json:"VAR,omitempty"`
	Type         string `json:"Type,omitempty"`
}

// BasketballTotalScore aggregates basketball period scores.
type BasketballTotalScore struct {
	Points int32 `json:"Points"`
}

// BasketballFixtureScore is the live basketball scoreboard payload.
type BasketballFixtureScore struct {
	Participant1 BasketballTotalScore `json:"Participant1"`
	Participant2 BasketballTotalScore `json:"Participant2"`
}

// UsFootballTotalScore aggregates American football scoring.
type UsFootballTotalScore struct {
	Points int32 `json:"Points"`
}

// UsFootballFixtureScore is the live American football scoreboard payload.
type UsFootballFixtureScore struct {
	Participant1 UsFootballTotalScore `json:"Participant1"`
	Participant2 UsFootballTotalScore `json:"Participant2"`
}

// UsFootballFixtureClock is the running match clock for American football.
type UsFootballFixtureClock struct {
	Running bool  `json:"running"`
	Seconds int32 `json:"seconds"`
}
