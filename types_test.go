package palrest

import (
	"encoding/json"
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
