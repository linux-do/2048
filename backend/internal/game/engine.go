package game

import (
	"math/rand"
	"time"
	"game2048/pkg/models"
)

// Engine handles the core 2048 game logic
type Engine struct {
	rng *rand.Rand
}

// NewEngine creates a new game engine
func NewEngine() *Engine {
	return &Engine{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// NewGame creates a new game with initial tiles
func (e *Engine) NewGame() models.Board {
	board := models.NewBoard()
	
	// Add two initial tiles
	e.addRandomTile(&board)
	e.addRandomTile(&board)
	
	return board
}

// Move executes a move in the given direction and returns the new board and score gained
func (e *Engine) Move(board models.Board, direction models.Direction) (models.Board, int, bool) {
	newBoard := board.Copy()
	scoreGained := 0
	moved := false
	
	switch direction {
	case models.DirectionUp:
		scoreGained, moved = e.moveUp(&newBoard)
	case models.DirectionDown:
		scoreGained, moved = e.moveDown(&newBoard)
	case models.DirectionLeft:
		scoreGained, moved = e.moveLeft(&newBoard)
	case models.DirectionRight:
		scoreGained, moved = e.moveRight(&newBoard)
	}
	
	// Add a new tile if the move was valid
	if moved {
		e.addRandomTile(&newBoard)
	}
	
	return newBoard, scoreGained, moved
}

// IsGameOver checks if the game is over (no valid moves available)
func (e *Engine) IsGameOver(board models.Board) bool {
	// If there are empty cells, game is not over
	if !board.IsFull() {
		return false
	}
	
	// Check if any moves are possible
	directions := []models.Direction{
		models.DirectionUp, models.DirectionDown,
		models.DirectionLeft, models.DirectionRight,
	}
	
	for _, dir := range directions {
		_, _, moved := e.Move(board, dir)
		if moved {
			return false
		}
	}
	
	return true
}

// IsVictory checks if the player has achieved victory (8192 tile)
func (e *Engine) IsVictory(board models.Board) bool {
	return board.HasVictoryTile()
}

// addRandomTile adds a random tile (2 or 4) to an empty position
func (e *Engine) addRandomTile(board *models.Board) bool {
	emptyCells := board.GetEmptyCells()
	if len(emptyCells) == 0 {
		return false
	}
	
	// Choose random empty cell
	pos := emptyCells[e.rng.Intn(len(emptyCells))]
	
	// 90% chance for 2, 10% chance for 4
	value := 2
	if e.rng.Float32() < 0.1 {
		value = 4
	}
	
	board.SetCell(pos[0], pos[1], value)
	return true
}

// moveLeft moves all tiles to the left and merges them
func (e *Engine) moveLeft(board *models.Board) (int, bool) {
	scoreGained := 0
	moved := false
	
	for row := 0; row < models.BoardSize; row++ {
		// Extract non-zero values
		var line []int
		for col := 0; col < models.BoardSize; col++ {
			if board.GetCell(row, col) != 0 {
				line = append(line, board.GetCell(row, col))
			}
		}
		
		// Merge adjacent equal values
		merged := e.mergeLine(line)
		scoreGained += merged.score
		
		// Check if anything changed
		for col := 0; col < models.BoardSize; col++ {
			newValue := 0
			if col < len(merged.line) {
				newValue = merged.line[col]
			}
			
			if board.GetCell(row, col) != newValue {
				moved = true
			}
			board.SetCell(row, col, newValue)
		}
	}
	
	return scoreGained, moved
}

// moveRight moves all tiles to the right
func (e *Engine) moveRight(board *models.Board) (int, bool) {
	scoreGained := 0
	moved := false
	
	for row := 0; row < models.BoardSize; row++ {
		// Extract non-zero values (in reverse order)
		var line []int
		for col := models.BoardSize - 1; col >= 0; col-- {
			if board.GetCell(row, col) != 0 {
				line = append(line, board.GetCell(row, col))
			}
		}
		
		// Merge adjacent equal values
		merged := e.mergeLine(line)
		scoreGained += merged.score
		
		// Place back in reverse order
		for col := 0; col < models.BoardSize; col++ {
			newValue := 0
			if col < len(merged.line) {
				newValue = merged.line[col]
			}
			
			actualCol := models.BoardSize - 1 - col
			if board.GetCell(row, actualCol) != newValue {
				moved = true
			}
			board.SetCell(row, actualCol, newValue)
		}
	}
	
	return scoreGained, moved
}

// moveUp moves all tiles up
func (e *Engine) moveUp(board *models.Board) (int, bool) {
	scoreGained := 0
	moved := false
	
	for col := 0; col < models.BoardSize; col++ {
		// Extract non-zero values
		var line []int
		for row := 0; row < models.BoardSize; row++ {
			if board.GetCell(row, col) != 0 {
				line = append(line, board.GetCell(row, col))
			}
		}
		
		// Merge adjacent equal values
		merged := e.mergeLine(line)
		scoreGained += merged.score
		
		// Check if anything changed
		for row := 0; row < models.BoardSize; row++ {
			newValue := 0
			if row < len(merged.line) {
				newValue = merged.line[row]
			}
			
			if board.GetCell(row, col) != newValue {
				moved = true
			}
			board.SetCell(row, col, newValue)
		}
	}
	
	return scoreGained, moved
}

// moveDown moves all tiles down
func (e *Engine) moveDown(board *models.Board) (int, bool) {
	scoreGained := 0
	moved := false
	
	for col := 0; col < models.BoardSize; col++ {
		// Extract non-zero values (in reverse order)
		var line []int
		for row := models.BoardSize - 1; row >= 0; row-- {
			if board.GetCell(row, col) != 0 {
				line = append(line, board.GetCell(row, col))
			}
		}
		
		// Merge adjacent equal values
		merged := e.mergeLine(line)
		scoreGained += merged.score
		
		// Place back in reverse order
		for row := 0; row < models.BoardSize; row++ {
			newValue := 0
			if row < len(merged.line) {
				newValue = merged.line[row]
			}
			
			actualRow := models.BoardSize - 1 - row
			if board.GetCell(actualRow, col) != newValue {
				moved = true
			}
			board.SetCell(actualRow, col, newValue)
		}
	}
	
	return scoreGained, moved
}

// mergeResult represents the result of merging a line
type mergeResult struct {
	line  []int
	score int
}

// mergeLine merges adjacent equal values in a line
func (e *Engine) mergeLine(line []int) mergeResult {
	if len(line) <= 1 {
		return mergeResult{line: line, score: 0}
	}
	
	var result []int
	score := 0
	i := 0
	
	for i < len(line) {
		if i+1 < len(line) && line[i] == line[i+1] {
			// Merge the two tiles
			merged := line[i] * 2
			result = append(result, merged)
			score += merged
			i += 2 // Skip both tiles
		} else {
			// Keep the tile as is
			result = append(result, line[i])
			i++
		}
	}
	
	return mergeResult{line: result, score: score}
}
