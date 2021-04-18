package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/gofrs/flock"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
)

type fsm struct {
	dataFile string
	lock     *flock.Flock
}

type fsmSnapshot struct {
	data []byte
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	log.Info("fsm.Apply called", "type", hclog.Fmt("%d", l.Type), "data", hclog.Fmt("%s", l.Data))

	var cmd Command
	if err := json.Unmarshal(l.Data, &cmd); err != nil {
		log.Error("failed command unmarshal", "error", err)
		return nil
	}

	ctx := context.Background()
	switch cmd.Action {
	case "set":
		return f.localSet(ctx, cmd.Key, cmd.Value)
	case "delete":
		return f.localDelete(ctx, cmd.Key)
	default:
		log.Error("unknown command", "command", cmd, "log", l)
	}

	return nil
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	log.Info("fsm.Snapshot called")

	data, err := f.loadData(context.Background())
	if err != nil {
		return nil, err
	}

	encodedData, err := encode(data)
	if err != nil {
		return nil, err
	}

	return &fsmSnapshot{data: encodedData}, nil
}

func (f *fsm) Restore(old io.ReadCloser) error {
	log.Info("fs.Restore called")
	b, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	data, err := decode(b)
	if err != nil {
		return err
	}

	return f.saveData(context.Background(), data)
}

func (f *fsm) localSet(ctx context.Context, key, value string) error {
	data, err := f.loadData(ctx)
	if err != nil {
		return err
	}

	data[key] = value
	return f.saveData(ctx, data)
}

// Get gets the value at the specified key
func (f *fsm) localGet(ctx context.Context, key string) (string, error) {
	data, err := f.loadData(ctx)
	if err != nil {
		return "", err
	}

	return data[key], nil
}

func (f *fsm) localDelete(ctx context.Context, key string) error {
	data, err := f.loadData(ctx)
	if err != nil {
		return err
	}

	delete(data, key)

	return f.saveData(ctx, data)
}

func (f *fsm) loadData(ctx context.Context) (map[string]string, error) {
	empty := map[string]string{}

	if f.lock == nil {
		f.lock = flock.New(f.dataFile)
	}
	defer f.lock.Close()

	locked, err := f.lock.TryLockContext(ctx, time.Microsecond)
	if err != nil {
		return empty, fmt.Errorf("trylock: %w", err)
	}

	if locked {
		// First check if the folder exists and create it if it is missing
		if _, err := os.Stat(f.dataFile); os.IsNotExist(err) {
			emptyData, err := encode(map[string]string{})
			if err != nil {
				return empty, fmt.Errorf("encode: %w", err)
			}

			if err := ioutil.WriteFile(f.dataFile, emptyData, 0644); err != nil {
				return empty, fmt.Errorf("write: %w", err)
			}
		}

		content, err := ioutil.ReadFile(f.dataFile)
		if err != nil {
			return empty, fmt.Errorf("read file: %w", err)
		}

		return decode(content)
	}

	return empty, fmt.Errorf("couldn't get lock")

}

func (f *fsm) saveData(ctx context.Context, data map[string]string) error {
	encodedData, err := encode(data)
	if err != nil {
		return err
	}

	if f.lock == nil {
		f.lock = flock.New(f.dataFile)
	}
	defer f.lock.Close()

	locked, err := f.lock.TryLockContext(ctx, time.Microsecond)
	if err != nil {
		return err
	}

	if locked {
		if err := ioutil.WriteFile(f.dataFile, encodedData, 0644); err != nil {
			return err
		}

		if err := f.lock.Unlock(); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("couldn't get lock")
}

func encode(data map[string]string) ([]byte, error) {
	encodedData := map[string]string{}
	for k, v := range data {
		ek := base64.URLEncoding.EncodeToString([]byte(k))
		ev := base64.URLEncoding.EncodeToString([]byte(v))
		encodedData[ek] = ev
	}

	return json.Marshal(encodedData)
}

func decode(data []byte) (map[string]string, error) {
	var jsonData map[string]string

	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, err
	}

	returnData := map[string]string{}
	for k, v := range jsonData {
		dk, err := base64.URLEncoding.DecodeString(k)
		if err != nil {
			return nil, err
		}

		dv, err := base64.URLEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}

		returnData[string(dk)] = string(dv)
	}

	return returnData, nil
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	log.Info("fsmSnapshot.Persist called")
	if _, err := sink.Write(s.data); err != nil {
		return err
	}
	defer sink.Close()

	return nil
}

func (s *fsmSnapshot) Release() {
	log.Info("fsmSnapshot.Release called")
}
