package palrest

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestActor_UnmarshalJSON_Null(t *testing.T) {
	var actor Actor

	dec := json.NewDecoder(strings.NewReader("null"))
	if err := dec.Decode(&actor); err != nil {
		t.Fatalf("decoding null actor failed: %v", err)
	}
	if actor.Type != "" || actor.Character != nil || actor.PalBox != nil {
		t.Fatalf("null actor should stay zeroed: %+v", actor)
	}
}

func TestActor_UnmarshalJSON_MissingType(t *testing.T) {
	var actor Actor

	dec := json.NewDecoder(strings.NewReader(`{"InstanceID":"char-1"}`))
	if err := dec.Decode(&actor); err == nil {
		t.Fatal("expected error for actor without Type discriminator")
	}
}

func TestActor_UnmarshalJSON_NullTypeRejected(t *testing.T) {
	var actor Actor

	dec := json.NewDecoder(strings.NewReader(`{"Type":null,"InstanceID":"char-1"}`))
	if err := dec.Decode(&actor); err == nil {
		t.Fatal("expected error for null Type discriminator")
	}
}

func TestActor_UnmarshalJSON_EmptyTypeRejected(t *testing.T) {
	var actor Actor

	dec := json.NewDecoder(strings.NewReader(`{"Type":""}`))
	if err := dec.Decode(&actor); err == nil {
		t.Fatal("expected error for empty Type discriminator")
	}
}

func TestActor_UnmarshalJSON_NonObjectRejected(t *testing.T) {
	var actor Actor

	dec := json.NewDecoder(strings.NewReader(`123`))
	if err := dec.Decode(&actor); err == nil {
		t.Fatal("expected error for a non-object actor payload")
	}
}

func TestActor_UnmarshalJSON_CharacterDecodeError(t *testing.T) {
	var actor Actor

	dec := json.NewDecoder(strings.NewReader(`{"Type":"Character","HP":"abc"}`))
	if err := dec.Decode(&actor); err == nil {
		t.Fatal("expected error for a character actor with a wrong field type")
	}
}

func TestActor_UnmarshalJSON_PalBoxDecodeError(t *testing.T) {
	var actor Actor

	dec := json.NewDecoder(strings.NewReader(`{"Type":"PalBox","GuildID":123}`))
	if err := dec.Decode(&actor); err == nil {
		t.Fatal("expected error for a pal box actor with a wrong field type")
	}
}

func TestActor_UnmarshalJSON_NullResetsState(t *testing.T) {
	actor := Actor{}

	dec := json.NewDecoder(strings.NewReader(`{"Type":"Character","InstanceID":"char-1"}`))
	if err := dec.Decode(&actor); err != nil {
		t.Fatalf("first decode failed: %v", err)
	}

	dec = json.NewDecoder(strings.NewReader(`null`))
	if err := dec.Decode(&actor); err != nil {
		t.Fatalf("null decode failed: %v", err)
	}
	if actor.Type != "" || actor.Character != nil || actor.PalBox != nil {
		t.Fatalf("null actor should reset all fields: %+v", actor)
	}
}

func TestActor_UnmarshalJSON_ResetsPointers(t *testing.T) {
	actor := Actor{}

	dec := json.NewDecoder(strings.NewReader(`{"Type":"Character","InstanceID":"char-1"}`))
	if err := dec.Decode(&actor); err != nil {
		t.Fatalf("first decode failed: %v", err)
	}
	if actor.Character == nil {
		t.Fatal("expected character actor after first decode")
	}

	dec = json.NewDecoder(strings.NewReader(`{"Type":"FutureKind","Whatever":1}`))
	if err := dec.Decode(&actor); err != nil {
		t.Fatalf("second decode failed: %v", err)
	}
	if actor.Character != nil || actor.PalBox != nil {
		t.Fatalf("stale pointers after unknown type: %+v", actor)
	}

	dec = json.NewDecoder(strings.NewReader(`{"Type":"PalBox","GuildID":"guild-2"}`))
	if err := dec.Decode(&actor); err != nil {
		t.Fatalf("third decode failed: %v", err)
	}
	if actor.PalBox == nil {
		t.Fatal("expected pal box actor after third decode")
	}
}

func TestActor_MarshalJSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		actor Actor
	}{
		{
			name: "character",
			actor: Actor{
				Type: "Character",
				Character: &CharacterActor{
					Type:       "Character",
					InstanceID: "char-1",
					UnitType:   "Player",
					NickName:   "PalUser",
					Level:      42,
					LocationX:  1.5,
					IsActive:   "true",
				},
			},
		},
		{
			name: "pal box",
			actor: Actor{
				Type: "PalBox",
				PalBox: &PalBoxActor{
					Type:      "PalBox",
					GuildID:   "guild-1",
					GuildName: "TestGuild",
					LocationY: 2.5,
				},
			},
		},
		{
			name:  "unknown type",
			actor: Actor{Type: "FutureKind"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.actor)
			if err != nil {
				t.Fatalf("marshal failed: %v", err)
			}
			var got Actor
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal failed: %v (json: %s)", err, data)
			}
			if !reflect.DeepEqual(got, tt.actor) {
				t.Fatalf("round-trip mismatch: got %+v, want %+v (json: %s)", got, tt.actor, data)
			}
		})
	}
}

func TestActor_MarshalJSON_ZeroIsNull(t *testing.T) {
	data, err := json.Marshal(Actor{})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(data) != "null" {
		t.Fatalf("unexpected zero actor JSON: %s", data)
	}
}

func TestActor_MarshalJSON_EmptyPayloads(t *testing.T) {
	tests := []struct {
		name  string
		actor Actor
		want  string
	}{
		{"character nil", Actor{Type: "Character"}, "Character"},
		{"pal box nil", Actor{Type: "PalBox"}, "PalBox"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertActorMarshalType(t, tt.actor, tt.want)
		})
	}
}

func TestActor_MarshalJSON_FillsType(t *testing.T) {
	assertActorMarshalType(t, Actor{Type: "Character", Character: &CharacterActor{}}, "Character")
	assertActorMarshalType(t, Actor{Type: "PalBox", PalBox: &PalBoxActor{}}, "PalBox")
}

func assertActorMarshalType(t *testing.T, actor Actor, want string) {
	t.Helper()

	data, err := json.Marshal(actor)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var got Actor
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.Type != want {
		t.Fatalf("unexpected type: %s", got.Type)
	}
	switch want {
	case "Character":
		if got.Character == nil || got.Character.Type != want {
			t.Fatalf("expected Character type to be filled: %s", data)
		}
	case "PalBox":
		if got.PalBox == nil || got.PalBox.Type != want {
			t.Fatalf("expected PalBox type to be filled: %s", data)
		}
	}
}
