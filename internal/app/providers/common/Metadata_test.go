package common

import (
	"reflect"
	"testing"
)

const (
	testUserAlice    = "alice"
	testTagEnvProd   = "prod"
	testTagVersion10 = "1.0"
	testImageName    = "postgres"
)

func TestStringFromMap(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "key exists with string value",
			m:    map[string]interface{}{"user": testUserAlice},
			key:  "user",
			want: testUserAlice,
		},
		{
			name: "key does not exist",
			m:    map[string]interface{}{"user": testUserAlice},
			key:  "email",
			want: "",
		},
		{
			name: "empty map",
			m:    map[string]interface{}{},
			key:  "user",
			want: "",
		},
		{
			name: "empty string value",
			m:    map[string]interface{}{"user": ""},
			key:  "user",
			want: "",
		},
		{
			name: "key with spaces value",
			m:    map[string]interface{}{"message": "initial commit"},
			key:  "message",
			want: "initial commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringFromMap(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("stringFromMap(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}

func TestMetadata_ToMap_Empty(t *testing.T) {
	m := Metadata{}
	result := m.ToMap()
	if len(result) != 0 {
		t.Errorf("ToMap() on empty Metadata should return empty map, got %v", result)
	}
}

func TestMetadata_ToMap_WithUserFields(t *testing.T) {
	m := Metadata{}
	m.SetUser(testUserAlice)
	m.SetEmail("alice@example.com")
	m.SetMessage("test commit")
	m.SetSource("local")

	result := m.ToMap()

	if result["user"] != testUserAlice {
		t.Errorf("ToMap() user = %v, want %q", result["user"], testUserAlice)
	}
	if result["email"] != "alice@example.com" {
		t.Errorf("ToMap() email = %v, want %q", result["email"], "alice@example.com")
	}
	if result["message"] != "test commit" {
		t.Errorf("ToMap() message = %v, want %q", result["message"], "test commit")
	}
	if result["source"] != "local" {
		t.Errorf("ToMap() source = %v, want %q", result["source"], "local")
	}
}

func TestMetadata_ToMap_OmitsEmptyFields(t *testing.T) {
	m := Metadata{}
	m.SetUser(testUserAlice)
	// email, message, source not set

	result := m.ToMap()

	if _, ok := result["email"]; ok {
		t.Error("ToMap() should not include empty email")
	}
	if _, ok := result["message"]; ok {
		t.Error("ToMap() should not include empty message")
	}
	if _, ok := result["source"]; ok {
		t.Error("ToMap() should not include empty source")
	}
	if _, ok := result["tags"]; ok {
		t.Error("ToMap() should not include empty tags")
	}
	if _, ok := result["timestamp"]; ok {
		t.Error("ToMap() should not include empty timestamp")
	}
}

func TestMetadata_ToMap_WithTags(t *testing.T) {
	m := Metadata{}
	m.SetTags(map[string]string{"env": testTagEnvProd, "version": testTagVersion10})

	result := m.ToMap()

	tags, ok := result["tags"].(map[string]string)
	if !ok {
		t.Fatalf("ToMap() tags should be map[string]string, got %T", result["tags"])
	}
	if tags["env"] != testTagEnvProd {
		t.Errorf("ToMap() tags[env] = %q, want %q", tags["env"], testTagEnvProd)
	}
	if tags["version"] != testTagVersion10 {
		t.Errorf("ToMap() tags[version] = %q, want %q", tags["version"], testTagVersion10)
	}
}

func TestMetadata_ToMap_V2Format(t *testing.T) {
	m := Metadata{
		version: V2,
		user:    testUserAlice,
		image: image{
			Image:  testImageName,
			Tag:    "latest",
			Digest: "sha256:abc123",
		},
		ports: []port{
			{Protocol: "tcp", Port: "5432"},
		},
		volumes: []volume{
			{Name: "data", Path: "/var/lib/postgresql/data"},
		},
		privileged:     false,
		disablePortMap: true,
	}

	result := m.ToMap()

	v2, ok := result["v2"].(map[string]interface{})
	if !ok {
		t.Fatalf("ToMap() should include v2 key with map value, got %T", result["v2"])
	}

	img, ok := v2["image"].(image)
	if !ok {
		t.Fatalf("v2[image] should be image type, got %T", v2["image"])
	}
	if img.Image != testImageName {
		t.Errorf("v2 image.Image = %q, want %q", img.Image, testImageName)
	}
	if img.Tag != "latest" {
		t.Errorf("v2 image.Tag = %q, want %q", img.Tag, "latest")
	}

	if v2["privileged"] != false {
		t.Errorf("v2 privileged = %v, want false", v2["privileged"])
	}
	if v2["disablePortMap"] != true {
		t.Errorf("v2 disablePortMap = %v, want true", v2["disablePortMap"])
	}

	// Should not have V1 fields
	if _, ok := result["container"]; ok {
		t.Error("V2 format should not include 'container' key")
	}
}

func TestMetadata_ToMap_V1Format(t *testing.T) {
	m := Metadata{
		version: V1,
		user:    "bob",
		image: image{
			Image:  "mysql",
			Tag:    "8.0",
			Digest: "mysql@sha256:def456",
		},
	}

	result := m.ToMap()

	if result["container"] != "mysql@sha256:def456" {
		t.Errorf("V1 container = %v, want %q", result["container"], "mysql@sha256:def456")
	}
	if result["image"] != "mysql" {
		t.Errorf("V1 image = %v, want %q", result["image"], "mysql")
	}
	if result["tag"] != "8.0" {
		t.Errorf("V1 tag = %v, want %q", result["tag"], "8.0")
	}
	if result["digest"] != "mysql@sha256:def456" {
		t.Errorf("V1 digest = %v, want %q", result["digest"], "mysql@sha256:def456")
	}

	// Should not have V2 fields
	if _, ok := result["v2"]; ok {
		t.Error("V1 format should not include 'v2' key")
	}
}

func TestMetadata_Load_DetectsV2(t *testing.T) {
	metaMap := map[string]interface{}{
		"user":    testUserAlice,
		"email":   "alice@example.com",
		"message": "test",
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  testImageName,
				"tag":    "latest",
				"digest": "sha256:abc",
			},
			"environment": nil,
			"ports":       nil,
			"volumes":     []interface{}{},
			"privileged":  true,
		},
	}

	m := Metadata{}
	loaded := m.Load(metaMap)

	if loaded.version != V2 {
		t.Errorf("Load() should detect V2 format, got version=%q", loaded.version)
	}
	if loaded.user != testUserAlice {
		t.Errorf("Load() user = %q, want %q", loaded.user, testUserAlice)
	}
	if loaded.privileged != true {
		t.Errorf("Load() privileged = %v, want true", loaded.privileged)
	}
}

func TestMetadata_Load_DetectsV1(t *testing.T) {
	metaMap := map[string]interface{}{
		"user":      "bob",
		"email":     "bob@example.com",
		"message":   "v1 commit",
		"container": "postgres@sha256:abc",
	}

	m := Metadata{}
	loaded := m.Load(metaMap)

	if loaded.version != V1 {
		t.Errorf("Load() should detect V1 format, got version=%q", loaded.version)
	}
	if loaded.user != "bob" {
		t.Errorf("Load() user = %q, want %q", loaded.user, "bob")
	}
}

func TestMetadata_MapV2_FullParse(t *testing.T) {
	metaMap := map[string]interface{}{
		"user":      testUserAlice,
		"email":     "alice@example.com",
		"message":   "snapshot",
		"source":    "local",
		"timestamp": "2026-01-15T10:30:00Z",
		"tags": map[string]interface{}{
			"env":     "staging",
			"version": "2.0",
		},
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  testImageName,
				"tag":    "16",
				"digest": "sha256:abc123",
			},
			"environment": []interface{}{"POSTGRES_PASSWORD=secret", "PGDATA=/data"},
			"ports": []interface{}{
				map[string]interface{}{
					"protocol": "tcp",
					"port":     "5432",
				},
			},
			"volumes": []interface{}{
				map[string]interface{}{
					"name": "pgdata",
					"path": "/var/lib/postgresql/data",
				},
			},
			"privileged":     false,
			"disablePortMap": true,
		},
	}

	m := Metadata{}
	result := m.MapV2(metaMap)

	if result.version != V2 {
		t.Errorf("MapV2() version = %q, want %q", result.version, V2)
	}
	if result.user != testUserAlice {
		t.Errorf("MapV2() user = %q, want %q", result.user, testUserAlice)
	}
	if result.email != "alice@example.com" {
		t.Errorf("MapV2() email = %q, want %q", result.email, "alice@example.com")
	}
	if result.message != "snapshot" {
		t.Errorf("MapV2() message = %q, want %q", result.message, "snapshot")
	}
	if result.source != "local" {
		t.Errorf("MapV2() source = %q, want %q", result.source, "local")
	}
	if result.timestamp != "2026-01-15T10:30:00Z" {
		t.Errorf("MapV2() timestamp = %q, want %q", result.timestamp, "2026-01-15T10:30:00Z")
	}
	if result.tags["env"] != "staging" {
		t.Errorf("MapV2() tags[env] = %q, want %q", result.tags["env"], "staging")
	}
	if result.tags["version"] != "2.0" {
		t.Errorf("MapV2() tags[version] = %q, want %q", result.tags["version"], "2.0")
	}
	if result.image.Image != testImageName {
		t.Errorf("MapV2() image.Image = %q, want %q", result.image.Image, testImageName)
	}
	if result.image.Tag != "16" {
		t.Errorf("MapV2() image.Tag = %q, want %q", result.image.Tag, "16")
	}
	if result.image.Digest != "sha256:abc123" {
		t.Errorf("MapV2() image.Digest = %q, want %q", result.image.Digest, "sha256:abc123")
	}
	if len(result.environment) != 2 {
		t.Errorf("MapV2() environment length = %d, want 2", len(result.environment))
	}
	if len(result.ports) != 1 {
		t.Errorf("MapV2() ports length = %d, want 1", len(result.ports))
	}
	if result.ports[0].Protocol != "tcp" || result.ports[0].Port != "5432" {
		t.Errorf("MapV2() ports[0] = %+v, want {tcp 5432}", result.ports[0])
	}
	if len(result.volumes) != 1 {
		t.Errorf("MapV2() volumes length = %d, want 1", len(result.volumes))
	}
	if result.volumes[0].Name != "pgdata" || result.volumes[0].Path != "/var/lib/postgresql/data" {
		t.Errorf("MapV2() volumes[0] = %+v, want {pgdata /var/lib/postgresql/data}", result.volumes[0])
	}
	if result.privileged != false {
		t.Errorf("MapV2() privileged = %v, want false", result.privileged)
	}
	if result.disablePortMap != true {
		t.Errorf("MapV2() disablePortMap = %v, want true", result.disablePortMap)
	}
}

func TestMetadata_MapV2_NilEnvironment(t *testing.T) {
	metaMap := map[string]interface{}{
		"user": testUserAlice,
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  testImageName,
				"tag":    "latest",
				"digest": "sha256:abc",
			},
			"environment": nil,
			"ports":       nil,
			"volumes":     []interface{}{},
		},
	}

	m := Metadata{}
	result := m.MapV2(metaMap)

	if result.environment != nil {
		t.Errorf("MapV2() with nil environment should have nil environment, got %v", result.environment)
	}
	if result.ports != nil {
		t.Errorf("MapV2() with nil ports should have nil ports, got %v", result.ports)
	}
}

func TestMetadata_MapV2_NoTags(t *testing.T) {
	metaMap := map[string]interface{}{
		"user": testUserAlice,
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  testImageName,
				"tag":    "latest",
				"digest": "sha256:abc",
			},
			"environment": nil,
			"ports":       nil,
			"volumes":     []interface{}{},
		},
	}

	m := Metadata{}
	result := m.MapV2(metaMap)

	if result.tags != nil {
		t.Errorf("MapV2() with no tags should have nil tags, got %v", result.tags)
	}
}

func TestMetadata_MapV2_PrivilegedDefaults(t *testing.T) {
	metaMap := map[string]interface{}{
		"user": testUserAlice,
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  testImageName,
				"tag":    "latest",
				"digest": "sha256:abc",
			},
			"environment": nil,
			"ports":       nil,
			"volumes":     []interface{}{},
			// privileged and disablePortMap intentionally omitted
		},
	}

	m := Metadata{}
	result := m.MapV2(metaMap)

	if result.privileged != false {
		t.Errorf("MapV2() with missing privileged should default to false, got %v", result.privileged)
	}
	if result.disablePortMap != false {
		t.Errorf("MapV2() with missing disablePortMap should default to false, got %v", result.disablePortMap)
	}
}

func TestMetadata_MapV1_BasicParse(t *testing.T) {
	metaMap := map[string]interface{}{
		"user":      "bob",
		"email":     "bob@example.com",
		"message":   "initial",
		"source":    "remote",
		"timestamp": "2026-01-10T08:00:00Z",
		"container": "postgres@sha256:abc123",
	}

	m := Metadata{}
	result := m.MapV1(metaMap)

	if result.version != V1 {
		t.Errorf("MapV1() version = %q, want %q", result.version, V1)
	}
	if result.user != "bob" {
		t.Errorf("MapV1() user = %q, want %q", result.user, "bob")
	}
	if result.email != "bob@example.com" {
		t.Errorf("MapV1() email = %q, want %q", result.email, "bob@example.com")
	}
	if result.image.Digest != "postgres@sha256:abc123" {
		t.Errorf("MapV1() image.Digest = %q, want %q", result.image.Digest, "postgres@sha256:abc123")
	}
	// When Image key is not present, image name is derived from digest
	if result.image.Image != testImageName {
		t.Errorf("MapV1() image.Image = %q, want %q (derived from digest with @)", result.image.Image, testImageName)
	}
}

func TestMetadata_MapV1_DigestWithColonFallback(t *testing.T) {
	metaMap := map[string]interface{}{
		"container": "postgres:latest",
	}

	m := Metadata{}
	result := m.MapV1(metaMap)

	// When digest contains : but not @, image name is derived by splitting on :
	if result.image.Image != testImageName {
		t.Errorf("MapV1() image.Image = %q, want %q (derived from digest with :)", result.image.Image, testImageName)
	}
}

func TestMetadata_MapV1_ExplicitImageAndTag(t *testing.T) {
	metaMap := map[string]interface{}{
		"container": "postgres@sha256:abc123",
		"Image":     "custom-postgres",
		"Tag":       "16-alpine",
	}

	m := Metadata{}
	result := m.MapV1(metaMap)

	if result.image.Image != "custom-postgres" {
		t.Errorf("MapV1() image.Image = %q, want %q", result.image.Image, "custom-postgres")
	}
	if result.image.Tag != "16-alpine" {
		t.Errorf("MapV1() image.Tag = %q, want %q", result.image.Tag, "16-alpine")
	}
}

func TestMetadata_MapV1_TagsAsMapInterface(t *testing.T) {
	metaMap := map[string]interface{}{
		"container": "postgres:latest",
		"tags": map[string]interface{}{
			"env":     testTagEnvProd,
			"version": testTagVersion10,
		},
	}

	m := Metadata{}
	result := m.MapV1(metaMap)

	if result.tags["env"] != testTagEnvProd {
		t.Errorf("MapV1() tags[env] = %q, want %q", result.tags["env"], testTagEnvProd)
	}
	if result.tags["version"] != testTagVersion10 {
		t.Errorf("MapV1() tags[version] = %q, want %q", result.tags["version"], testTagVersion10)
	}
}

func TestMetadata_MapV1_TagsAsArray(t *testing.T) {
	metaMap := map[string]interface{}{
		"container": "postgres:latest",
		"tags":      []interface{}{"env:prod", "version:1.0", "standalone"},
	}

	m := Metadata{}
	result := m.MapV1(metaMap)

	if result.tags["env"] != testTagEnvProd {
		t.Errorf("MapV1() tags[env] = %q, want %q", result.tags["env"], testTagEnvProd)
	}
	if result.tags["version"] != testTagVersion10 {
		t.Errorf("MapV1() tags[version] = %q, want %q", result.tags["version"], testTagVersion10)
	}
	// Standalone tag (no colon) should have empty value
	if val, ok := result.tags["standalone"]; !ok || val != "" {
		t.Errorf("MapV1() tags[standalone] = %q, want %q", val, "")
	}
}

func TestMetadata_MapV1_TagsUnsupportedType(t *testing.T) {
	metaMap := map[string]interface{}{
		"container": "postgres:latest",
		"tags":      42, // unsupported type
	}

	m := Metadata{}
	result := m.MapV1(metaMap)

	if result.tags != nil {
		t.Errorf("MapV1() with unsupported tags type should have nil tags, got %v", result.tags)
	}
}

func TestMetadata_MapV1_RuntimeParsing(t *testing.T) {
	metaMap := map[string]interface{}{
		"container": "postgres:latest",
		"runtime":   "[-e POSTGRES_PASSWORD=secret -p 5432:5432]",
	}

	m := Metadata{}
	result := m.MapV1(metaMap)

	if len(result.environment) != 1 {
		t.Fatalf("MapV1() environment length = %d, want 1", len(result.environment))
	}
	if result.environment[0] != "POSTGRES_PASSWORD=secret" {
		t.Errorf("MapV1() environment[0] = %v, want %q", result.environment[0], "POSTGRES_PASSWORD=secret")
	}
	if len(result.ports) != 1 {
		t.Fatalf("MapV1() ports length = %d, want 1", len(result.ports))
	}
	if result.ports[0].Port != "5432" {
		t.Errorf("MapV1() ports[0].Port = %q, want %q", result.ports[0].Port, "5432")
	}
	if result.ports[0].Protocol != "tcp" {
		t.Errorf("MapV1() ports[0].Protocol = %q, want %q", result.ports[0].Protocol, "tcp")
	}
}

func TestMetadata_GetPrivileged(t *testing.T) {
	tests := []struct {
		name       string
		privileged bool
		want       bool
	}{
		{"privileged true", true, true},
		{"privileged false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Metadata{privileged: tt.privileged}
			if got := m.GetPrivileged(); got != tt.want {
				t.Errorf("GetPrivileged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadata_GetDisablePortMap(t *testing.T) {
	tests := []struct {
		name           string
		disablePortMap bool
		want           bool
	}{
		{"disablePortMap true", true, true},
		{"disablePortMap false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Metadata{disablePortMap: tt.disablePortMap}
			if got := m.GetDisablePortMap(); got != tt.want {
				t.Errorf("GetDisablePortMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadata_SettersAndToMap_Roundtrip(t *testing.T) {
	m := Metadata{}
	m.SetUser("charlie")
	m.SetEmail("charlie@example.com")
	m.SetMessage("test roundtrip")
	m.SetSource("s3")
	m.SetTags(map[string]string{"release": "v1.0"})

	result := m.ToMap()

	if result["user"] != "charlie" {
		t.Errorf("roundtrip user = %v, want %q", result["user"], "charlie")
	}
	if result["email"] != "charlie@example.com" {
		t.Errorf("roundtrip email = %v, want %q", result["email"], "charlie@example.com")
	}
	if result["message"] != "test roundtrip" {
		t.Errorf("roundtrip message = %v, want %q", result["message"], "test roundtrip")
	}
	if result["source"] != "s3" {
		t.Errorf("roundtrip source = %v, want %q", result["source"], "s3")
	}
	tags, ok := result["tags"].(map[string]string)
	if !ok {
		t.Fatalf("roundtrip tags type = %T, want map[string]string", result["tags"])
	}
	if !reflect.DeepEqual(tags, map[string]string{"release": "v1.0"}) {
		t.Errorf("roundtrip tags = %v, want %v", tags, map[string]string{"release": "v1.0"})
	}
}

func TestMetadata_MapV2_MultiplePorts(t *testing.T) {
	metaMap := map[string]interface{}{
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  "myapp",
				"tag":    "latest",
				"digest": "sha256:xyz",
			},
			"environment": nil,
			"ports": []interface{}{
				map[string]interface{}{"protocol": "tcp", "port": "8080"},
				map[string]interface{}{"protocol": "tcp", "port": "8443"},
				map[string]interface{}{"protocol": "udp", "port": "9090"},
			},
			"volumes": []interface{}{},
		},
	}

	m := Metadata{}
	result := m.MapV2(metaMap)

	if len(result.ports) != 3 {
		t.Fatalf("MapV2() ports length = %d, want 3", len(result.ports))
	}
	expectedPorts := []port{
		{Protocol: "tcp", Port: "8080"},
		{Protocol: "tcp", Port: "8443"},
		{Protocol: "udp", Port: "9090"},
	}
	for i, ep := range expectedPorts {
		if result.ports[i].Protocol != ep.Protocol || result.ports[i].Port != ep.Port {
			t.Errorf("MapV2() ports[%d] = %+v, want %+v", i, result.ports[i], ep)
		}
	}
}

func TestMetadata_MapV2_MultipleVolumes(t *testing.T) {
	metaMap := map[string]interface{}{
		"v2": map[string]interface{}{
			"image": map[string]interface{}{
				"image":  "myapp",
				"tag":    "latest",
				"digest": "sha256:xyz",
			},
			"environment": nil,
			"ports":       nil,
			"volumes": []interface{}{
				map[string]interface{}{"name": "data", "path": "/data"},
				map[string]interface{}{"name": "logs", "path": "/var/log"},
			},
		},
	}

	m := Metadata{}
	result := m.MapV2(metaMap)

	if len(result.volumes) != 2 {
		t.Fatalf("MapV2() volumes length = %d, want 2", len(result.volumes))
	}
	if result.volumes[0].Name != "data" || result.volumes[0].Path != "/data" {
		t.Errorf("MapV2() volumes[0] = %+v, want {data /data}", result.volumes[0])
	}
	if result.volumes[1].Name != "logs" || result.volumes[1].Path != "/var/log" {
		t.Errorf("MapV2() volumes[1] = %+v, want {logs /var/log}", result.volumes[1])
	}
}
