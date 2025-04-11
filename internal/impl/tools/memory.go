package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aiagent/internal/domain/entities"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// KnowledgeGraph represents the structure stored in MongoDB
type KnowledgeGraph struct {
	Entities  []Entity   `bson:"entities"`
	Relations []Relation `bson:"relations"`
}

// Entity represents a node in the knowledge graph
type Entity struct {
	Name         string   `json:"name" bson:"name"`
	EntityType   string   `json:"entityType" bson:"entityType"`
	Observations []string `json:"observations" bson:"observations"`
}

// Relation represents an edge between entities
type Relation struct {
	From         string `json:"from" bson:"from"`
	To           string `json:"to" bson:"to"`
	RelationType string `json:"relationType" bson:"relationType"`
}

type MemoryTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	collection    *mongo.Collection
}

func NewMemoryTool(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
	// Expect MongoDB collection to be passed in configuration
	collectionName, ok := configuration["mongo_collection"]
	if !ok {
		logger.Error("mongo_collection not specified in configuration")
		collectionName = "knowledge_graph"
	}

	// Assume MongoDB client is initialized elsewhere (in main.go)
	// For this tool, we'll use the same database as other repositories
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(configuration["mongo_uri"]))
	if err != nil {
		logger.Error("Failed to connect to MongoDB", zap.Error(err))
	}
	db := client.Database("aiagent")
	collection := db.Collection(collectionName)

	return &MemoryTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
		collection:    collection,
	}
}

func (t *MemoryTool) Name() string {
	return t.name
}

func (t *MemoryTool) Description() string {
	return t.description
}

func (t *MemoryTool) Configuration() map[string]string {
	return t.configuration
}

func (t *MemoryTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *MemoryTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"create_entities", "create_relations", "add_observations", "delete_entities", "delete_observations", "delete_relations", "read_graph", "search_nodes", "open_nodes"},
			Description: "The memory operation to perform",
			Required:    true,
		},
		{
			Name:        "entities",
			Type:        "array",
			Items:       []entities.Item{{Type: "object"}},
			Description: "Array of entities with name, entityType, and observations (for create_entities)",
			Required:    false,
		},
		{
			Name:        "relations",
			Type:        "array",
			Items:       []entities.Item{{Type: "object"}},
			Description: "Array of relations with from, to, and relationType (for create_relations, delete_relations)",
			Required:    false,
		},
		{
			Name:        "observations",
			Type:        "array",
			Items:       []entities.Item{{Type: "object"}},
			Description: "Array of observations with entityName and contents (for add_observations)",
			Required:    false,
		},
		{
			Name:        "entityNames",
			Type:        "array",
			Items:       []entities.Item{{Type: "string"}},
			Description: "Array of entity names (for delete_entities)",
			Required:    false,
		},
		{
			Name:        "deletions",
			Type:        "array",
			Items:       []entities.Item{{Type: "object"}},
			Description: "Array of deletions with entityName and observations (for delete_observations)",
			Required:    false,
		},
		{
			Name:        "query",
			Type:        "string",
			Description: "Search query for nodes (for search_nodes)",
			Required:    false,
		},
		{
			Name:        "names",
			Type:        "array",
			Items:       []entities.Item{{Type: "string"}},
			Description: "Array of entity names to retrieve (for open_nodes)",
			Required:    false,
		},
	}
}

func (t *MemoryTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing memory operation", zap.String("arguments", arguments))

	var args struct {
		Operation    string                   `json:"operation"`
		Entities     []Entity                 `json:"entities"`
		Relations    []Relation               `json:"relations"`
		Observations []map[string]interface{} `json:"observations"`
		EntityNames  []string                 `json:"entityNames"`
		Deletions    []map[string]interface{} `json:"deletions"`
		Query        string                   `json:"query"`
		Names        []string                 `json:"names"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	if args.Operation == "" {
		t.logger.Error("Operation is required")
		return "", fmt.Errorf("operation is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch args.Operation {
	case "create_entities":
		return t.createEntities(ctx, args.Entities)
	case "create_relations":
		return t.createRelations(ctx, args.Relations)
	case "add_observations":
		return t.addObservations(ctx, args.Observations)
	case "delete_entities":
		return t.deleteEntities(ctx, args.EntityNames)
	case "delete_observations":
		return t.deleteObservations(ctx, args.Deletions)
	case "delete_relations":
		return t.deleteRelations(ctx, args.Relations)
	case "read_graph":
		return t.readGraph(ctx)
	case "search_nodes":
		return t.searchNodes(ctx, args.Query)
	case "open_nodes":
		return t.openNodes(ctx, args.Names)
	default:
		t.logger.Error("Unknown operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unknown operation: %s", args.Operation)
	}
}

func (t *MemoryTool) loadGraph(ctx context.Context) (*KnowledgeGraph, error) {
	var graph KnowledgeGraph
	err := t.collection.FindOne(ctx, bson.M{"_id": "graph"}).Decode(&graph)
	if err == mongo.ErrNoDocuments {
		return &KnowledgeGraph{Entities: []Entity{}, Relations: []Relation{}}, nil
	}
	if err != nil {
		t.logger.Error("Failed to load graph", zap.Error(err))
		return nil, err
	}
	return &graph, nil
}

func (t *MemoryTool) saveGraph(ctx context.Context, graph *KnowledgeGraph) error {
	_, err := t.collection.UpdateOne(
		ctx,
		bson.M{"_id": "graph"},
		bson.M{"$set": graph},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		t.logger.Error("Failed to save graph", zap.Error(err))
		return err
	}
	return nil
}

func (t *MemoryTool) createEntities(ctx context.Context, entities []Entity) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	var newEntities []Entity
	for _, entity := range entities {
		if !containsEntity(graph.Entities, entity.Name) {
			newEntities = append(newEntities, entity)
			graph.Entities = append(graph.Entities, entity)
		}
	}

	if err := t.saveGraph(ctx, graph); err != nil {
		return "", err
	}

	result, _ := json.MarshalIndent(newEntities, "", "  ")
	t.logger.Info("Entities created", zap.Int("count", len(newEntities)))
	return string(result), nil
}

func (t *MemoryTool) createRelations(ctx context.Context, relations []Relation) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	var newRelations []Relation
	for _, relation := range relations {
		if !containsRelation(graph.Relations, relation) {
			newRelations = append(newRelations, relation)
			graph.Relations = append(graph.Relations, relation)
		}
	}

	if err := t.saveGraph(ctx, graph); err != nil {
		return "", err
	}

	result, _ := json.MarshalIndent(newRelations, "", "  ")
	t.logger.Info("Relations created", zap.Int("count", len(newRelations)))
	return string(result), nil
}

func (t *MemoryTool) addObservations(ctx context.Context, observations []map[string]interface{}) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	var results []map[string]interface{}
	for _, obs := range observations {
		entityName, _ := obs["entityName"].(string)
		contents, _ := obs["contents"].([]interface{})
		if entityName == "" || len(contents) == 0 {
			continue
		}

		var stringContents []string
		for _, c := range contents {
			if s, ok := c.(string); ok {
				stringContents = append(stringContents, s)
			}
		}

		for i, entity := range graph.Entities {
			if entity.Name == entityName {
				var added []string
				for _, content := range stringContents {
					if !containsString(entity.Observations, content) {
						added = append(added, content)
						graph.Entities[i].Observations = append(graph.Entities[i].Observations, content)
					}
				}
				results = append(results, map[string]interface{}{
					"entityName":        entityName,
					"addedObservations": added,
				})
				break
			}
		}
	}

	if err := t.saveGraph(ctx, graph); err != nil {
		return "", err
	}

	result, _ := json.MarshalIndent(results, "", "  ")
	t.logger.Info("Observations added", zap.Int("count", len(results)))
	return string(result), nil
}

func (t *MemoryTool) deleteEntities(ctx context.Context, entityNames []string) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	var newEntities []Entity
	for _, entity := range graph.Entities {
		if !containsString(entityNames, entity.Name) {
			newEntities = append(newEntities, entity)
		}
	}

	var newRelations []Relation
	for _, relation := range graph.Relations {
		if !containsString(entityNames, relation.From) && !containsString(entityNames, relation.To) {
			newRelations = append(newRelations, relation)
		}
	}

	graph.Entities = newEntities
	graph.Relations = newRelations

	if err := t.saveGraph(ctx, graph); err != nil {
		return "", err
	}

	t.logger.Info("Entities deleted", zap.Int("count", len(entityNames)))
	return "Entities deleted successfully", nil
}

func (t *MemoryTool) deleteObservations(ctx context.Context, deletions []map[string]interface{}) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	for _, del := range deletions {
		entityName, _ := del["entityName"].(string)
		observations, _ := del["observations"].([]interface{})
		var stringObs []string
		for _, o := range observations {
			if s, ok := o.(string); ok {
				stringObs = append(stringObs, s)
			}
		}

		for i, entity := range graph.Entities {
			if entity.Name == entityName {
				var newObs []string
				for _, obs := range entity.Observations {
					if !containsString(stringObs, obs) {
						newObs = append(newObs, obs)
					}
				}
				graph.Entities[i].Observations = newObs
			}
		}
	}

	if err := t.saveGraph(ctx, graph); err != nil {
		return "", err
	}

	t.logger.Info("Observations deleted")
	return "Observations deleted successfully", nil
}

func (t *MemoryTool) deleteRelations(ctx context.Context, relations []Relation) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	var newRelations []Relation
	for _, existing := range graph.Relations {
		shouldKeep := true
		for _, toDelete := range relations {
			if existing.From == toDelete.From && existing.To == toDelete.To && existing.RelationType == toDelete.RelationType {
				shouldKeep = false
				break
			}
		}
		if shouldKeep {
			newRelations = append(newRelations, existing)
		}
	}

	graph.Relations = newRelations

	if err := t.saveGraph(ctx, graph); err != nil {
		return "", err
	}

	t.logger.Info("Relations deleted", zap.Int("count", len(relations)))
	return "Relations deleted successfully", nil
}

func (t *MemoryTool) readGraph(ctx context.Context) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	result, _ := json.MarshalIndent(graph, "", "  ")
	t.logger.Info("Graph read successfully")
	return string(result), nil
}

func (t *MemoryTool) searchNodes(ctx context.Context, query string) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	query = strings.ToLower(query)
	var filteredEntities []Entity
	for _, entity := range graph.Entities {
		if strings.Contains(strings.ToLower(entity.Name), query) ||
			strings.Contains(strings.ToLower(entity.EntityType), query) ||
			containsMatchingString(entity.Observations, query) {
			filteredEntities = append(filteredEntities, entity)
		}
	}

	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}

	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	filteredGraph := KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}

	result, _ := json.MarshalIndent(filteredGraph, "", "  ")
	t.logger.Info("Nodes searched", zap.String("query", query))
	return string(result), nil
}

func (t *MemoryTool) openNodes(ctx context.Context, names []string) (string, error) {
	graph, err := t.loadGraph(ctx)
	if err != nil {
		return "", err
	}

	var filteredEntities []Entity
	for _, entity := range graph.Entities {
		if containsString(names, entity.Name) {
			filteredEntities = append(filteredEntities, entity)
		}
	}

	filteredEntityNames := make(map[string]bool)
	for _, entity := range filteredEntities {
		filteredEntityNames[entity.Name] = true
	}

	var filteredRelations []Relation
	for _, relation := range graph.Relations {
		if filteredEntityNames[relation.From] && filteredEntityNames[relation.To] {
			filteredRelations = append(filteredRelations, relation)
		}
	}

	filteredGraph := KnowledgeGraph{
		Entities:  filteredEntities,
		Relations: filteredRelations,
	}

	result, _ := json.MarshalIndent(filteredGraph, "", "  ")
	t.logger.Info("Nodes opened", zap.Int("count", len(names)))
	return string(result), nil
}

// Helper functions
func containsEntity(entities []Entity, name string) bool {
	for _, entity := range entities {
		if entity.Name == name {
			return true
		}
	}
	return false
}

func containsRelation(relations []Relation, relation Relation) bool {
	for _, r := range relations {
		if r.From == relation.From && r.To == relation.To && r.RelationType == relation.RelationType {
			return true
		}
	}
	return false
}

func containsString(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

func containsMatchingString(slice []string, query string) bool {
	query = strings.ToLower(query)
	for _, s := range slice {
		if strings.Contains(strings.ToLower(s), query) {
			return true
		}
	}
	return false
}

var _ entities.Tool = (*MemoryTool)(nil)
