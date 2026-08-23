package palrest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// ServerInfo contains basic information returned by the /v1/api/info endpoint.
type ServerInfo struct {
	Version     string `json:"version"`
	ServerName  string `json:"servername"`
	Description string `json:"description"`
	WorldGUID   string `json:"worldguid"`
}

// PlayerInfo represents a player connected to the server.
type PlayerInfo struct {
	Name          string  `json:"name"`
	AccountName   string  `json:"accountName"`
	PlayerID      string  `json:"playerId"`
	UserID        string  `json:"userId"`
	IP            string  `json:"ip"`
	Ping          float64 `json:"ping"`
	LocationX     float64 `json:"location_x"`
	LocationY     float64 `json:"location_y"`
	Level         int     `json:"level"`
	BuildingCount int     `json:"building_count"`
}

// PlayerList groups the list of players returned by the /v1/api/players endpoint.
type PlayerList struct {
	Players []PlayerInfo `json:"players"`
}

// ServerSettings contains the server settings returned by the /v1/api/settings
// endpoint. Fields follow the official REST API schema; numeric settings are
// floats and toggles are booleans.
type ServerSettings struct {
	Difficulty                                      string   `json:"Difficulty"`
	RandomizerType                                  string   `json:"RandomizerType"`
	RandomizerSeed                                  string   `json:"RandomizerSeed"`
	IsRandomizerPalLevelRandom                      bool     `json:"bIsRandomizerPalLevelRandom"`
	DayTimeSpeedRate                                float64  `json:"DayTimeSpeedRate"`
	NightTimeSpeedRate                              float64  `json:"NightTimeSpeedRate"`
	ExpRate                                         float64  `json:"ExpRate"`
	PalCaptureRate                                  float64  `json:"PalCaptureRate"`
	PalSpawnNumRate                                 float64  `json:"PalSpawnNumRate"`
	PalDamageRateAttack                             float64  `json:"PalDamageRateAttack"`
	PalDamageRateDefense                            float64  `json:"PalDamageRateDefense"`
	PlayerDamageRateAttack                          float64  `json:"PlayerDamageRateAttack"`
	PlayerDamageRateDefense                         float64  `json:"PlayerDamageRateDefense"`
	PlayerStomachDecreaceRate                       float64  `json:"PlayerStomachDecreaceRate"`
	PlayerStaminaDecreaceRate                       float64  `json:"PlayerStaminaDecreaceRate"`
	PlayerAutoHPRegeneRate                          float64  `json:"PlayerAutoHPRegeneRate"`
	PlayerAutoHpRegeneRateInSleep                   float64  `json:"PlayerAutoHpRegeneRateInSleep"`
	PalStomachDecreaceRate                          float64  `json:"PalStomachDecreaceRate"`
	PalStaminaDecreaceRate                          float64  `json:"PalStaminaDecreaceRate"`
	PalAutoHPRegeneRate                             float64  `json:"PalAutoHPRegeneRate"`
	PalAutoHpRegeneRateInSleep                      float64  `json:"PalAutoHpRegeneRateInSleep"`
	BuildObjectHpRate                               float64  `json:"BuildObjectHpRate"`
	BuildObjectDamageRate                           float64  `json:"BuildObjectDamageRate"`
	BuildObjectDeteriorationDamageRate              float64  `json:"BuildObjectDeteriorationDamageRate"`
	CollectionDropRate                              float64  `json:"CollectionDropRate"`
	CollectionObjectHpRate                          float64  `json:"CollectionObjectHpRate"`
	CollectionObjectRespawnSpeedRate                float64  `json:"CollectionObjectRespawnSpeedRate"`
	EnemyDropItemRate                               float64  `json:"EnemyDropItemRate"`
	DeathPenalty                                    string   `json:"DeathPenalty"`
	EnablePlayerToPlayerDamage                      bool     `json:"bEnablePlayerToPlayerDamage"`
	EnableFriendlyFire                              bool     `json:"bEnableFriendlyFire"`
	EnableInvaderEnemy                              bool     `json:"bEnableInvaderEnemy"`
	ActiveUNKO                                      bool     `json:"bActiveUNKO"`
	EnableAimAssistPad                              bool     `json:"bEnableAimAssistPad"`
	EnableAimAssistKeyboard                         bool     `json:"bEnableAimAssistKeyboard"`
	DropItemMaxNum                                  float64  `json:"DropItemMaxNum"`
	PhysicsActiveDropItemMaxNum                     float64  `json:"PhysicsActiveDropItemMaxNum"`
	DropItemMaxNumUNKO                              float64  `json:"DropItemMaxNum_UNKO"`
	BaseCampMaxNum                                  float64  `json:"BaseCampMaxNum"`
	BaseCampWorkerMaxNum                            float64  `json:"BaseCampWorkerMaxNum"`
	DropItemAliveMaxHours                           float64  `json:"DropItemAliveMaxHours"`
	AutoResetGuildNoOnlinePlayers                   bool     `json:"bAutoResetGuildNoOnlinePlayers"`
	AutoResetGuildTimeNoOnlinePlayers               float64  `json:"AutoResetGuildTimeNoOnlinePlayers"`
	GuildPlayerMaxNum                               float64  `json:"GuildPlayerMaxNum"`
	BaseCampMaxNumInGuild                           float64  `json:"BaseCampMaxNumInGuild"`
	PalEggDefaultHatchingTime                       float64  `json:"PalEggDefaultHatchingTime"`
	WorkSpeedRate                                   float64  `json:"WorkSpeedRate"`
	AutoSaveSpan                                    float64  `json:"autoSaveSpan"`
	IsMultiplay                                     bool     `json:"bIsMultiplay"`
	IsPvP                                           bool     `json:"bIsPvP"`
	Hardcore                                        bool     `json:"bHardcore"`
	PalLost                                         bool     `json:"bPalLost"`
	CharacterRecreateInHardcore                     bool     `json:"bCharacterRecreateInHardcore"`
	CanPickupOtherGuildDeathPenaltyDrop             bool     `json:"bCanPickupOtherGuildDeathPenaltyDrop"`
	EnableNonLoginPenalty                           bool     `json:"bEnableNonLoginPenalty"`
	EnableFastTravel                                bool     `json:"bEnableFastTravel"`
	EnableFastTravelOnlyBaseCamp                    bool     `json:"bEnableFastTravelOnlyBaseCamp"`
	IsStartLocationSelectByMap                      bool     `json:"bIsStartLocationSelectByMap"`
	ExistPlayerAfterLogout                          bool     `json:"bExistPlayerAfterLogout"`
	EnableDefenseOtherGuildPlayer                   bool     `json:"bEnableDefenseOtherGuildPlayer"`
	InvisibleOtherGuildBaseCampAreaFX               bool     `json:"bInvisibleOtherGuildBaseCampAreaFX"`
	BuildAreaLimit                                  bool     `json:"bBuildAreaLimit"`
	ItemWeightRate                                  float64  `json:"ItemWeightRate"`
	CoopPlayerMaxNum                                float64  `json:"CoopPlayerMaxNum"`
	ServerPlayerMaxNum                              float64  `json:"ServerPlayerMaxNum"`
	ServerName                                      string   `json:"ServerName"`
	ServerDescription                               string   `json:"ServerDescription"`
	AllowClientMod                                  bool     `json:"bAllowClientMod"`
	PublicPort                                      float64  `json:"PublicPort"`
	PublicIP                                        string   `json:"PublicIP"`
	RCONEnabled                                     bool     `json:"RCONEnabled"`
	RCONPort                                        float64  `json:"RCONPort"`
	Region                                          string   `json:"Region"`
	UseAuth                                         bool     `json:"bUseAuth"`
	BanListURL                                      string   `json:"BanListURL"`
	RESTAPIEnabled                                  bool     `json:"RESTAPIEnabled"`
	RESTAPIPort                                     float64  `json:"RESTAPIPort"`
	ShowPlayerList                                  bool     `json:"bShowPlayerList"`
	ChatPostLimitPerMinute                          float64  `json:"ChatPostLimitPerMinute"`
	CrossplayPlatforms                              []string `json:"CrossplayPlatforms"`
	IsUseBackupSaveData                             bool     `json:"bIsUseBackupSaveData"`
	LogFormatType                                   string   `json:"LogFormatType"`
	IsShowJoinLeftMessage                           bool     `json:"bIsShowJoinLeftMessage"`
	SupplyDropSpan                                  float64  `json:"SupplyDropSpan"`
	EnablePredatorBossPal                           bool     `json:"EnablePredatorBossPal"`
	MaxBuildingLimitNum                             float64  `json:"MaxBuildingLimitNum"`
	ServerReplicatePawnCullDistance                 float64  `json:"ServerReplicatePawnCullDistance"`
	AllowGlobalPalboxExport                         bool     `json:"bAllowGlobalPalboxExport"`
	AllowGlobalPalboxImport                         bool     `json:"bAllowGlobalPalboxImport"`
	EquipmentDurabilityDamageRate                   float64  `json:"EquipmentDurabilityDamageRate"`
	ItemContainerForceMarkDirtyInterval             float64  `json:"ItemContainerForceMarkDirtyInterval"`
	PlayerDataPalStorageUpdateCheckTickInterval     float64  `json:"PlayerDataPalStorageUpdateCheckTickInterval"`
	ItemCorruptionMultiplier                        float64  `json:"ItemCorruptionMultiplier"`
	MonsterFarmActionSpeedRate                      float64  `json:"MonsterFarmActionSpeedRate"`
	DenyTechnologyList                              []string `json:"DenyTechnologyList"`
	GuildRejoinCooldownMinutes                      float64  `json:"GuildRejoinCooldownMinutes"`
	AutoTransferMasterCheckIntervalSeconds          float64  `json:"AutoTransferMasterCheckIntervalSeconds"`
	AutoTransferMasterThresholdDays                 float64  `json:"AutoTransferMasterThresholdDays"`
	MaxGuildsPerFrame                               float64  `json:"MaxGuildsPerFrame"`
	BlockRespawnTime                                float64  `json:"BlockRespawnTime"`
	RespawnPenaltyDurationThreshold                 float64  `json:"RespawnPenaltyDurationThreshold"`
	RespawnPenaltyTimeScale                         float64  `json:"RespawnPenaltyTimeScale"`
	DisplayPvPItemNumOnWorldMap_BaseCamp            bool     `json:"bDisplayPvPItemNumOnWorldMap_BaseCamp"`
	DisplayPvPItemNumOnWorldMap_Player              bool     `json:"bDisplayPvPItemNumOnWorldMap_Player"`
	AdditionalDropItemWhenPlayerKillingInPvPMode    string   `json:"AdditionalDropItemWhenPlayerKillingInPvPMode"`
	AdditionalDropItemNumWhenPlayerKillingInPvPMode float64  `json:"AdditionalDropItemNumWhenPlayerKillingInPvPMode"`
	BAdditionalDropItemWhenPlayerKillingInPvPMode   bool     `json:"bAdditionalDropItemWhenPlayerKillingInPvPMode"`
	EnableVoiceChat                                 bool     `json:"bEnableVoiceChat"`
	VoiceChatMaxVolumeDistance                      float64  `json:"VoiceChatMaxVolumeDistance"`
	VoiceChatZeroVolumeDistance                     float64  `json:"VoiceChatZeroVolumeDistance"`
	AllowEnhanceStat_Health                         bool     `json:"bAllowEnhanceStat_Health"`
	AllowEnhanceStat_Attack                         bool     `json:"bAllowEnhanceStat_Attack"`
	AllowEnhanceStat_Stamina                        bool     `json:"bAllowEnhanceStat_Stamina"`
	AllowEnhanceStat_Weight                         bool     `json:"bAllowEnhanceStat_Weight"`
	AllowEnhanceStat_WorkSpeed                      bool     `json:"bAllowEnhanceStat_WorkSpeed"`
	EnableBuildingPlayerUIdDisplay                  bool     `json:"bEnableBuildingPlayerUIdDisplay"`
	BuildingNameDisplayCacheTTLSeconds              float64  `json:"BuildingNameDisplayCacheTTLSeconds"`
	AllowEnemyCampSpawnNearBaseCamp                 bool     `json:"bAllowEnemyCampSpawnNearBaseCamp"`
	AllowConnectPlatform                            string   `json:"AllowConnectPlatform"`
}

// ServerMetrics contains server performance metrics returned by the
// /v1/api/metrics endpoint.
type ServerMetrics struct {
	ServerFPS        int     `json:"serverfps"`
	CurrentPlayerNum int     `json:"currentplayernum"`
	ServerFrameTime  float64 `json:"serverframetime"`
	MaxPlayerNum     int     `json:"maxplayernum"`
	Uptime           int     `json:"uptime"`
	BaseCampNum      int     `json:"basecampnum"`
	Days             int     `json:"days"`
}

// GameData is the world actor snapshot returned by the /v1/api/game-data
// endpoint. Time uses the server-local "YYYY-MM-DD HH:MM:SS" format rather
// than ISO 8601; FPS is instantaneous and AverageFPS is the average server
// FPS. Requires the server to run with -enable-gamedata-api.
type GameData struct {
	Time       string  `json:"Time"`
	FPS        float64 `json:"FPS"`
	AverageFPS float64 `json:"AverageFPS"`
	ActorData  []Actor `json:"ActorData"`
}

// Actor is a single actor in the world snapshot. The server returns a
// discriminated union of Character and PalBox kinds (see CharacterActor and
// PalBoxActor); UnmarshalJSON populates the pointer matching the Type
// discriminator. Actor kinds introduced by future server versions keep Type
// set with both pointers nil.
type Actor struct {
	Type      string          `json:"Type"`
	Character *CharacterActor `json:"Character,omitempty"`
	PalBox    *PalBoxActor    `json:"PalBox,omitempty"`
}

// CharacterActor is an actor of type "Character": players, pals and NPCs.
// UnitType may be Player, OtomoPal, BaseCampPal, WildPal or NPC. Trainer
// fields apply to OtomoPal and BaseCampPal; UserID is player-only. Fields that
// do not apply to the UnitType are left as zero values.
type CharacterActor struct {
	Type              string  `json:"Type"`
	InstanceID        string  `json:"InstanceID"`
	UnitType          string  `json:"UnitType"`
	NickName          string  `json:"NickName"`
	TrainerInstanceID string  `json:"TrainerInstanceID"`
	TrainerNickName   string  `json:"TrainerNickName"`
	TrainerClass      string  `json:"TrainerClass"`
	UserID            string  `json:"userid"`
	IP                string  `json:"ip"`
	Level             int     `json:"level"`
	HP                int     `json:"HP"`
	MaxHP             int     `json:"MaxHP"`
	GuildID           string  `json:"GuildID"`
	GuildName         string  `json:"GuildName"`
	Class             string  `json:"Class"`
	Action            string  `json:"Action"`
	AIAction          string  `json:"AI_Action"`
	LocationX         float64 `json:"LocationX"`
	LocationY         float64 `json:"LocationY"`
	LocationZ         float64 `json:"LocationZ"`
	RotationX         float64 `json:"RotationX"`
	RotationY         float64 `json:"RotationY"`
	RotationZ         float64 `json:"RotationZ"`
	Stage             string  `json:"Stage"`
	IsActive          string  `json:"IsActive"` // API returns "true"/"false" JSON strings, not booleans
}

// PalBoxActor is an actor of type "PalBox": a guild base camp.
type PalBoxActor struct {
	Type      string  `json:"Type"`
	GuildID   string  `json:"GuildID"`
	GuildName string  `json:"GuildName"`
	Class     string  `json:"Class"`
	LocationX float64 `json:"LocationX"`
	LocationY float64 `json:"LocationY"`
	LocationZ float64 `json:"LocationZ"`
}

// UnmarshalJSON decodes the actor payload and populates the pointer matching
// the Type discriminator. Unknown non-empty Type values are kept on the Actor
// without an error to stay compatible with future server versions. Object
// payloads with a missing, null or empty Type are rejected; a JSON null actor
// entry is accepted and leaves the Actor zeroed so that it does not invalidate
// the whole snapshot.
func (a *Actor) UnmarshalJSON(data []byte) error {
	a.Character = nil
	a.PalBox = nil
	a.Type = ""

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}

	var header struct {
		Type string `json:"Type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return fmt.Errorf("failed to decode actor discriminator: %w", err)
	}
	if header.Type == "" {
		return errors.New("actor missing Type discriminator")
	}

	a.Type = header.Type
	switch header.Type {
	case "Character":
		a.Character = new(CharacterActor)
		if err := json.Unmarshal(data, a.Character); err != nil {
			return fmt.Errorf("failed to decode character actor: %w", err)
		}
	case "PalBox":
		a.PalBox = new(PalBoxActor)
		if err := json.Unmarshal(data, a.PalBox); err != nil {
			return fmt.Errorf("failed to decode pal box actor: %w", err)
		}
	}
	return nil
}
