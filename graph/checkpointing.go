package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// Checkpoint represents a saved state at a specific point in execution
type Checkpoint struct {
	ID        string                 `json:"id"`
	NodeName  string                 `json:"node_name"`
	State     interface{}            `json:"state"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	Version   int                    `json:"version"`
}

// CheckpointStore defines the interface for checkpoint persistence
type CheckpointStore interface {
	// Save stores a checkpoint
	Save(ctx context.Context, checkpoint *Checkpoint) error

	// Load retrieves a checkpoint by ID
	Load(ctx context.Context, checkpointID string) (*Checkpoint, error)

	// List returns all checkpoints for a given execution
	List(ctx context.Context, executionID string) ([]*Checkpoint, error)

	// Delete removes a checkpoint
	Delete(ctx context.Context, checkpointID string) error

	// Clear removes all checkpoints for an execution
	Clear(ctx context.Context, executionID string) error
}

// MemoryCheckpointStore provides in-memory checkpoint storage
type MemoryCheckpointStore struct {
	checkpoints map[string]*Checkpoint
	mutex       sync.RWMutex
}

// NewMemoryCheckpointStore creates a new in-memory checkpoint store
func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{
		checkpoints: make(map[string]*Checkpoint),
	}
}

// Save implements CheckpointStore interface
func (m *MemoryCheckpointStore) Save(_ context.Context, checkpoint *Checkpoint) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.checkpoints[checkpoint.ID] = checkpoint
	return nil
}

// Load implements CheckpointStore interface
func (m *MemoryCheckpointStore) Load(_ context.Context, checkpointID string) (*Checkpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	checkpoint, exists := m.checkpoints[checkpointID]
	if !exists {
		return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	return checkpoint, nil
}

// List implements CheckpointStore interface
func (m *MemoryCheckpointStore) List(_ context.Context, executionID string) ([]*Checkpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var checkpoints []*Checkpoint
	for _, checkpoint := range m.checkpoints {
		if execID, ok := checkpoint.Metadata["execution_id"].(string); ok && execID == executionID {
			checkpoints = append(checkpoints, checkpoint)
		}
	}

	// Sort by version (ascending order) so latest is last
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Version < checkpoints[j].Version
	})

	return checkpoints, nil
}

// Delete implements CheckpointStore interface
func (m *MemoryCheckpointStore) Delete(_ context.Context, checkpointID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.checkpoints, checkpointID)
	return nil
}

// Clear implements CheckpointStore interface
func (m *MemoryCheckpointStore) Clear(_ context.Context, executionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for id, checkpoint := range m.checkpoints {
		if execID, ok := checkpoint.Metadata["execution_id"].(string); ok && execID == executionID {
			delete(m.checkpoints, id)
		}
	}

	return nil
}

// FileCheckpointStore provides file-based checkpoint storage
type FileCheckpointStore struct {
	writer io.Writer
	reader io.Reader
	mutex  sync.RWMutex
}

// NewFileCheckpointStore creates a new file-based checkpoint store
func NewFileCheckpointStore(writer io.Writer, reader io.Reader) *FileCheckpointStore {
	return &FileCheckpointStore{
		writer: writer,
		reader: reader,
	}
}

// Save implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Save(_ context.Context, checkpoint *Checkpoint) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	data, err := json.Marshal(checkpoint)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	_, err = f.writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	return nil
}

// Load implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Load(_ context.Context, checkpointID string) (*Checkpoint, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	data, err := io.ReadAll(f.reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	var checkpoint Checkpoint
	err = json.Unmarshal(data, &checkpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	if checkpoint.ID != checkpointID {
		return nil, fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	return &checkpoint, nil
}

// List implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) List(_ context.Context, _ string) ([]*Checkpoint, error) {
	// For file storage, this would typically involve reading from multiple files
	// This is a simplified implementation
	return nil, fmt.Errorf("list operation not implemented for file store")
}

// Delete implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Delete(_ context.Context, checkpointID string) error {
	// For file storage, this would involve file system operations
	return fmt.Errorf("delete operation not implemented for file store")
}

// Clear implements CheckpointStore interface for file storage
func (f *FileCheckpointStore) Clear(_ context.Context, executionID string) error {
	// For file storage, this would involve file system operations
	return fmt.Errorf("clear operation not implemented for file store")
}

// CheckpointConfig configures checkpointing behavior
type CheckpointConfig struct {
	// Store is the checkpoint storage backend
	Store CheckpointStore

	// AutoSave enables automatic checkpointing after each node
	AutoSave bool

	// SaveInterval specifies how often to save (when AutoSave is false)
	SaveInterval time.Duration

	// MaxCheckpoints limits the number of checkpoints to keep
	MaxCheckpoints int
}

// DefaultCheckpointConfig returns a default checkpoint configuration
func DefaultCheckpointConfig() CheckpointConfig {
	return CheckpointConfig{
		Store:          NewMemoryCheckpointStore(),
		AutoSave:       true,
		SaveInterval:   30 * time.Second,
		MaxCheckpoints: 10,
	}
}

// CheckpointableRunnable wraps a runnable with checkpointing capabilities
type CheckpointableRunnable struct {
	runnable *ListenableRunnable
	config   CheckpointConfig

	executionID string
}

// NewCheckpointableRunnable creates a new checkpointable runnable
func NewCheckpointableRunnable(runnable *ListenableRunnable, config CheckpointConfig) *CheckpointableRunnable {
	return &CheckpointableRunnable{
		runnable:    runnable,
		config:      config,
		executionID: generateExecutionID(),
	}
}

// Invoke executes the graph with checkpointing
func (cr *CheckpointableRunnable) Invoke(ctx context.Context, initialState interface{}) (interface{}, error) {
	return cr.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig executes the graph with checkpointing and config
func (cr *CheckpointableRunnable) InvokeWithConfig(ctx context.Context, initialState interface{}, config *Config) (interface{}, error) {
	// Extract thread_id from config if present
	var threadID string
	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
	}

	// Create checkpointing listener
	checkpointListener := &CheckpointListener{
		store:       cr.config.Store,
		executionID: cr.executionID,
		threadID:    threadID,
		autoSave:    cr.config.AutoSave,
	}

	// Add checkpoint listener to config callbacks
	if config == nil {
		config = &Config{}
	}
	config.Callbacks = append(config.Callbacks, checkpointListener)

	return cr.runnable.InvokeWithConfig(ctx, initialState, config)
}

// SaveCheckpoint manually saves a checkpoint
func (cr *CheckpointableRunnable) SaveCheckpoint(ctx context.Context, nodeName string, state interface{}) error {
	// Get current version from existing checkpoints
	checkpoints, err := cr.config.Store.List(ctx, cr.executionID)
	version := 1
	if err == nil && len(checkpoints) > 0 {
		// Get the latest version
		latest := checkpoints[len(checkpoints)-1]
		version = latest.Version + 1
	}

	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  nodeName,
		State:     state,
		Timestamp: time.Now(),
		Version:   version,
		Metadata: map[string]interface{}{
			"execution_id": cr.executionID,
		},
	}

	return cr.config.Store.Save(ctx, checkpoint)
}

// LoadCheckpoint loads a specific checkpoint
func (cr *CheckpointableRunnable) LoadCheckpoint(ctx context.Context, checkpointID string) (*Checkpoint, error) {
	return cr.config.Store.Load(ctx, checkpointID)
}

// ListCheckpoints returns all checkpoints for this execution
func (cr *CheckpointableRunnable) ListCheckpoints(ctx context.Context) ([]*Checkpoint, error) {
	return cr.config.Store.List(ctx, cr.executionID)
}

// ResumeFromCheckpoint resumes execution from a specific checkpoint
func (cr *CheckpointableRunnable) ResumeFromCheckpoint(ctx context.Context, checkpointID string) (interface{}, error) {
	checkpoint, err := cr.LoadCheckpoint(ctx, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Resume execution from the checkpointed state
	// This would require the graph to support starting from a specific node
	// For now, we'll return the checkpointed state
	return checkpoint.State, nil
}

// ClearCheckpoints removes all checkpoints for this execution
func (cr *CheckpointableRunnable) ClearCheckpoints(ctx context.Context) error {
	return cr.config.Store.Clear(ctx, cr.executionID)
}

// CheckpointListener automatically creates checkpoints during execution
type CheckpointListener struct {
	store       CheckpointStore
	executionID string
	threadID    string
	autoSave    bool
	// Embed NoOpCallbackHandler to satisfy other CallbackHandler methods
	NoOpCallbackHandler
}

// OnGraphStep implements GraphCallbackHandler
func (cl *CheckpointListener) OnGraphStep(ctx context.Context, stepNode string, state interface{}) {
	if !cl.autoSave {
		return
	}

	// Get current version from existing checkpoints
	checkpoints, err := cl.store.List(ctx, cl.executionID)
	version := 1
	if err == nil && len(checkpoints) > 0 {
		// Get the latest version
		latest := checkpoints[len(checkpoints)-1]
		version = latest.Version + 1
	}

	metadata := map[string]interface{}{
		"execution_id": cl.executionID,
		"event":        "step",
	}
	if cl.threadID != "" {
		metadata["thread_id"] = cl.threadID
	}

	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  stepNode,
		State:     state,
		Timestamp: time.Now(),
		Version:   version,
		Metadata:  metadata,
	}

	// Save checkpoint synchronously to avoid race conditions in tests
	if saveErr := cl.store.Save(ctx, checkpoint); saveErr != nil {
		_ = saveErr
	}
}

// OnNodeEvent is no longer used for saving state, but kept if needed for interface compatibility
// or we can remove it if we don't use it as NodeListener anymore.
// CheckpointableRunnable currently adds it as NodeListener. We should change that.

// CheckpointableMessageGraph extends ListenableMessageGraph with checkpointing
type CheckpointableMessageGraph struct {
	*ListenableMessageGraph
	config CheckpointConfig
}

// NewCheckpointableMessageGraph creates a new checkpointable message graph
func NewCheckpointableMessageGraph() *CheckpointableMessageGraph {
	return &CheckpointableMessageGraph{
		ListenableMessageGraph: NewListenableMessageGraph(),
		config:                 DefaultCheckpointConfig(),
	}
}

// NewCheckpointableMessageGraphWithConfig creates a checkpointable graph with custom config
func NewCheckpointableMessageGraphWithConfig(config CheckpointConfig) *CheckpointableMessageGraph {
	return &CheckpointableMessageGraph{
		ListenableMessageGraph: NewListenableMessageGraph(),
		config:                 config,
	}
}

// CompileCheckpointable compiles the graph into a checkpointable runnable
func (g *CheckpointableMessageGraph) CompileCheckpointable() (*CheckpointableRunnable, error) {
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		return nil, err
	}

	return NewCheckpointableRunnable(listenableRunnable, g.config), nil
}

// SetCheckpointConfig updates the checkpointing configuration
func (g *CheckpointableMessageGraph) SetCheckpointConfig(config CheckpointConfig) {
	g.config = config
}

// GetCheckpointConfig returns the current checkpointing configuration
func (g *CheckpointableMessageGraph) GetCheckpointConfig() CheckpointConfig {
	return g.config
}

// Helper functions
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

func generateCheckpointID() string {
	return fmt.Sprintf("checkpoint_%d", time.Now().UnixNano())
}

// StateSnapshot represents a snapshot of the graph state
type StateSnapshot struct {
	Values    interface{}
	Next      []string
	Config    Config
	Metadata  map[string]interface{}
	CreatedAt time.Time
	ParentID  string
}

// GetState retrieves the state for the given config
func (cr *CheckpointableRunnable) GetState(ctx context.Context, config *Config) (*StateSnapshot, error) {
	var threadID string
	var checkpointID string

	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
		if cid, ok := config.Configurable["checkpoint_id"].(string); ok {
			checkpointID = cid
		}
	}

	// Default to current execution ID if thread_id not provided
	if threadID == "" {
		threadID = cr.executionID
	}

	var checkpoint *Checkpoint
	var err error

	if checkpointID != "" {
		checkpoint, err = cr.config.Store.Load(ctx, checkpointID)
	} else {
		// Get latest checkpoint for the thread
		// Note: List returns all checkpoints. We need to find the latest one.
		// This is inefficient for large histories. Real implementations should have GetLatest.
		checkpoints, err := cr.config.Store.List(ctx, threadID)
		if err == nil && len(checkpoints) > 0 {
			// Assuming List returns in some order, or we sort.
			// For now, assume the last one is latest (based on implementation of MemoryStore)
			checkpoint = checkpoints[len(checkpoints)-1]
		}
	}

	if err != nil {
		return nil, err
	}

	if checkpoint == nil {
		return &StateSnapshot{
			Values: nil,
			Config: *config,
		}, nil
	}

	// Construct snapshot
	snapshot := &StateSnapshot{
		Values:    checkpoint.State,
		CreatedAt: checkpoint.Timestamp,
		Metadata:  checkpoint.Metadata,
		Config: Config{
			Configurable: map[string]interface{}{
				"thread_id":     threadID,
				"checkpoint_id": checkpoint.ID,
			},
		},
	}

	// Determine "Next" nodes
	// This is tricky without re-running the graph logic or storing it in the checkpoint.
	// For now, we might leave it empty or try to infer it if we stored it.
	// In the future, Checkpoint should store "Next" nodes.

	return snapshot, nil
}

// UpdateState updates the state for the given config
func (cr *CheckpointableRunnable) UpdateState(ctx context.Context, config *Config, values interface{}, asNode string) (*Config, error) {
	var threadID string
	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
	}

	if threadID == "" {
		threadID = cr.executionID
	}

	// 1. Get current state
	// We need to find the latest checkpoint for this thread to merge against
	checkpoints, err := cr.config.Store.List(ctx, threadID)
	var currentState interface{}
	var currentVersion int

	if err == nil && len(checkpoints) > 0 {
		// Assume last is latest
		latest := checkpoints[len(checkpoints)-1]
		currentState = latest.State
		currentVersion = latest.Version
	} else {
		// No existing state, initialize if schema exists
		if cr.runnable.graph.Schema != nil {
			currentState = cr.runnable.graph.Schema.Init()
		}
	}

	// 2. Merge values
	newState := values
	if cr.runnable.graph.Schema != nil {
		// If we have a current state, merge into it
		if currentState != nil {
			newState, err = cr.runnable.graph.Schema.Update(currentState, values)
			if err != nil {
				return nil, fmt.Errorf("failed to merge state: %w", err)
			}
		} else {
			// If no current state, maybe Init + Update?
			// Or just use values if it matches schema?
			// Let's try Init + Update
			initial := cr.runnable.graph.Schema.Init()
			newState, err = cr.runnable.graph.Schema.Update(initial, values)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize and merge state: %w", err)
			}
		}
	} else if currentState != nil {
		// No schema, but have current state.
		// If map, try to merge? Or just overwrite?
		// Without schema, we usually default to overwrite or simple merge if map.
		// Let's assume overwrite if no schema, or maybe simple map merge if both are maps.
		if curMap, ok := currentState.(map[string]interface{}); ok {
			if valMap, ok := values.(map[string]interface{}); ok {
				// Simple map merge
				merged := make(map[string]interface{})
				for k, v := range curMap {
					merged[k] = v
				}
				for k, v := range valMap {
					merged[k] = v
				}
				newState = merged
			}
		}
	}

	// 3. Create new checkpoint
	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  asNode, // The node that "made" this update
		State:     newState,
		Timestamp: time.Now(),
		Version:   currentVersion + 1,
		Metadata: map[string]interface{}{
			"execution_id": threadID,
			"source":       "update_state",
			"updated_by":   asNode,
		},
	}

	if err := cr.config.Store.Save(ctx, checkpoint); err != nil {
		return nil, err
	}

	return &Config{
		Configurable: map[string]interface{}{
			"thread_id":     threadID,
			"checkpoint_id": checkpoint.ID,
		},
	}, nil
}
