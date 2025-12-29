// Puzzle Rush main functionality
import { 
  initializeChessground, 
  resetBoard, 
  handleMove as handleMoveBoard,
  setupPlayerNames 
} from '../flashcard/board.js';
import { updateEvalBar } from '../flashcard/eval-bar.js';

let currentSession = null;
let currentCard = null;
let rushStartTime = null;
let timerInterval = null;
let cardStartTime = null;

// Get session data from page
const rushDataScript = document.getElementById('rush-data');
if (rushDataScript) {
  try {
    const data = JSON.parse(rushDataScript.textContent);
    if (data && data.session) {
      currentSession = data.session;
      rushStartTime = Date.now() - (currentSession.total_time_seconds * 1000);
    }
  } catch (e) {
    console.error('Failed to parse rush data:', e);
  }
}

function showScreen(screenId) {
  document.getElementById('selection-screen').classList.toggle('is-hidden', screenId !== 'selection');
  document.getElementById('active-rush-screen').classList.toggle('is-hidden', screenId !== 'active');
  document.getElementById('results-screen').classList.toggle('is-hidden', screenId !== 'results');
}

function updateMistakesIndicator() {
  if (!currentSession) return;
  
  const indicator = document.getElementById('mistakes-indicator');
  indicator.innerHTML = '';
  
  for (let i = 0; i < currentSession.mistakes_allowed; i++) {
    const dot = document.createElement('div');
    dot.className = 'mistake-dot' + (i < currentSession.mistakes_made ? ' used' : '');
    indicator.appendChild(dot);
  }
}

function updateStats() {
  if (!currentSession) return;
  
  document.getElementById('current-score').textContent = currentSession.score;
  
  const elapsed = Date.now() - rushStartTime;
  const seconds = Math.floor(elapsed / 1000);
  const minutes = Math.floor(seconds / 60);
  const displaySeconds = seconds % 60;
  document.getElementById('time-elapsed').textContent = `${minutes}:${displaySeconds.toString().padStart(2, '0')}`;
  
  updateMistakesIndicator();
}

function startTimer() {
  if (timerInterval) return;
  
  timerInterval = setInterval(() => {
    updateStats();
  }, 100);
}

function stopTimer() {
  if (timerInterval) {
    clearInterval(timerInterval);
    timerInterval = null;
  }
}

async function startRush(difficulty) {
  try {
    const formData = new FormData();
    formData.append('difficulty', difficulty);
    
    const response = await fetch('/puzzle-rush/start', {
      method: 'POST',
      body: formData
    });
    
    if (!response.ok) {
      const error = await response.json();
      alert('Failed to start rush: ' + (error.message || 'Unknown error'));
      return;
    }
    
    const data = await response.json();
    currentSession = data.session;
    currentCard = data.card;
    rushStartTime = Date.now();
    
    showScreen('active');
    updateStats();
    startTimer();
    
    if (currentCard) {
      loadFlashcard(currentCard);
    } else {
      alert('No flashcards available. Please add some games first.');
      showScreen('selection');
    }
  } catch (error) {
    console.error('Error starting rush:', error);
    alert('Failed to start rush. Please try again.');
  }
}

async function submitAnswer(flashcardId, quality, timeSeconds) {
  if (!currentSession) return;
  
  try {
    const formData = new FormData();
    formData.append('session_id', currentSession.id);
    formData.append('flashcard_id', flashcardId);
    formData.append('quality', quality);
    formData.append('time_seconds', timeSeconds);
    
    const response = await fetch('/puzzle-rush/answer', {
      method: 'POST',
      body: formData
    });
    
    if (!response.ok) {
      const error = await response.json();
      alert('Failed to submit answer: ' + (error.message || 'Unknown error'));
      return;
    }
    
    const data = await response.json();
    currentSession = data.session;
    
    // Check if session ended
    if (currentSession.completed_at) {
      stopTimer();
      showResults();
      return;
    }
    
    // Load next card
    if (data.nextCard) {
      currentCard = data.nextCard;
      loadFlashcard(currentCard);
    } else {
      alert('No more flashcards available.');
      showScreen('selection');
    }
  } catch (error) {
    console.error('Error submitting answer:', error);
    alert('Failed to submit answer. Please try again.');
  }
}

function loadFlashcard(card) {
  if (!card) return;
  
  const container = document.getElementById('flashcard-container');
  
  // Parse FEN
  let fen = card.fen;
  if (typeof fen === 'string') {
    fen = fen.replace(/^["']+|["']+$/g, '').trim();
  }
  
  const bestMove = card.best_move.trim();
  const mateBefore = card.mate_before;
  const mateAfter = card.mate_after;
  const evalBefore = parseFloat(card.eval_before) / 100;
  const evalAfter = parseFloat(card.eval_after) / 100;
  const evalDiff = parseFloat(card.eval_diff) / 100;
  const classification = card.classification.toLowerCase();
  const movePlayed = card.move_played;
  const prevMovePlayed = card.prev_move_played;
  const whitePlayer = card.white_player;
  const blackPlayer = card.black_player;
  
  // Parse last move
  let lastMove = null;
  if (prevMovePlayed && prevMovePlayed.length >= 4) {
    const from = prevMovePlayed.substring(0, 2);
    const to = prevMovePlayed.substring(2, 4);
    lastMove = [from, to];
  }
  
  // Create flashcard HTML
  container.innerHTML = `
    <div class="card rush-metadata-card">
      <div class="card-content">
        <div class="columns is-mobile is-multiline mb-0">
          <div class="column is-half-mobile">
            <p class="heading is-size-7 mb-1">Move</p>
            <p class="is-size-6 has-text-weight-semibold">#${card.move_number}</p>
          </div>
          <div class="column is-half-mobile">
            <p class="heading is-size-7 mb-1">Classification</p>
            <span class="tag is-${classification === 'blunder' ? 'danger' : classification === 'mistake' ? 'warning' : 'info'}">
              ${classification}
            </span>
          </div>
        </div>
        <div class="is-size-7 has-text-grey mt-2">
          ${whitePlayer} vs ${blackPlayer} • ${card.time_class}
        </div>
      </div>
    </div>
    
    <div class="rush-flashcard-layout">
      <div>
        <div class="board-container">
          <div class="player-info" id="player-top">
            <span class="piece-icon" id="icon-top"></span>
            <span class="player-name" id="name-top"></span>
          </div>
          <div class="board-area">
            <div class="eval-bar">
              <div class="eval-label white-advantage" id="eval-label">+0.0</div>
              <div class="eval-track">
                <div class="eval-fill" id="eval-fill"></div>
              </div>
            </div>
            <div class="board-wrapper">
              <div id="rush-board"></div>
            </div>
          </div>
          <div class="player-info" id="player-bottom">
            <span class="piece-icon" id="icon-bottom"></span>
            <span class="player-name" id="name-bottom"></span>
          </div>
        </div>
        <div class="mt-3">
          <button id="show-answer-rush" class="button is-light is-small">Show answer</button>
        </div>
      </div>
      
      <div class="feedback-section">
        <div class="feedback-box" id="feedback-box-rush">
          <div class="level is-mobile mb-2">
            <div class="level-left">
              <p id="prompt-text-rush" class="has-text-weight-semibold is-size-5">Find the best move in this position</p>
            </div>
          </div>
          <div id="feedback-content-rush">
            <p id="feedback-text-rush" class="mt-2"></p>
          </div>
        </div>
      </div>
    </div>
  `;
  
  // Initialize chess.js
  let chess;
  try {
    chess = new Chess(fen);
  } catch (e) {
    console.error('Failed to initialize Chess.js:', e);
    return;
  }
  
  const sideToMove = fen.split(" ")[1] === "w" ? "white" : "black";
  cardStartTime = Date.now();
  let isCompleted = false;
  let wasCorrect = false;
  
  function contextualPrompt() {
    const isUserWhite = sideToMove === "white";
    const userEval = isUserWhite ? evalBefore : -evalBefore;
    
    if (mateBefore !== null && mateBefore !== undefined) {
      const userMate = isUserWhite ? mateBefore : -mateBefore;
      if (userMate > 0) {
        return "Find the fastest mate";
      } else if (userMate < 0) {
        return "Find the only move to avoid mate";
      }
    }
    
    if (userEval > 2) return "Find the move to keep your advantage";
    if (userEval > 0.5) return "Maintain the pressure with the best move";
    if (userEval < -2) return "Find the move to stay in the game";
    if (userEval < -0.5) return "Find the best defensive resource";
    return "Find the best move in this balanced position";
  }
  
  function revealResult(isCorrect, moveUci) {
    if (isCompleted) return;
    
    wasCorrect = isCorrect;
    isCompleted = true;
    
    const timeSeconds = (Date.now() - cardStartTime) / 1000;
    
    // Determine quality (0-5)
    // For puzzle rush, we simplify: correct = 4 (Good), incorrect = 1 (Hard)
    const quality = isCorrect ? 4 : 1;
    
    const feedbackEl = document.getElementById('feedback-text-rush');
    const feedbackBox = document.getElementById('feedback-box-rush');
    
    if (isCorrect) {
      feedbackEl.textContent = "Correct! Great move.";
      feedbackEl.classList.remove("has-text-danger");
      feedbackEl.classList.add("has-text-success");
      feedbackBox.classList.remove("error");
      feedbackBox.classList.add("success");
    } else {
      feedbackEl.textContent = `Incorrect. Best move was ${bestMove}.`;
      feedbackEl.classList.remove("has-text-success");
      feedbackEl.classList.add("has-text-danger");
      feedbackBox.classList.remove("success");
      feedbackBox.classList.add("error");
      
      // Draw arrow for best move
      if (bestMove && bestMove.length >= 4) {
        const from = bestMove.substring(0, 2);
        const to = bestMove.substring(2, 4);
        cg.setShapes([{ orig: from, dest: to, brush: 'green' }]);
      }
    }
    
    // Disable board
    cg.set({ movable: { color: undefined } });
    
    // Submit answer after a short delay
    setTimeout(() => {
      submitAnswer(card.id, quality, timeSeconds);
    }, 1500);
  }
  
  async function handleMoveCallback(orig, dest) {
    if (isCompleted) return;
    
    const move = chess.move({ from: orig, to: dest, promotion: 'q' });
    if (!move) return;
    
    const moveUci = orig + dest;
    const isCorrect = moveUci.toLowerCase() === bestMove.toLowerCase();
    
    if (isCorrect) {
      revealResult(true, moveUci);
    } else {
      revealResult(false, moveUci);
    }
  }
  
  // Initialize board
  let cg;
  try {
    const boardEl = document.getElementById('rush-board');
    cg = initializeChessground(boardEl, fen, sideToMove, lastMove, chess, handleMoveCallback);
  } catch (err) {
    console.error('Failed to initialize board:', err);
    return;
  }
  
  // Show answer button
  document.getElementById('show-answer-rush').addEventListener('click', function() {
    if (isCompleted) return;
    wasCorrect = false;
    revealResult(false, '');
  });
  
  // Set prompt
  document.getElementById('prompt-text-rush').textContent = contextualPrompt();
  updateEvalBar(document.getElementById('eval-fill'), document.getElementById('eval-label'), evalBefore, mateBefore);
  setupPlayerNames(sideToMove, whitePlayer, blackPlayer);
  
  updateStats();
}

function showResults() {
  if (!currentSession) return;
  
  showScreen('results');
  
  document.getElementById('final-score').textContent = currentSession.score;
  document.getElementById('final-correct').textContent = currentSession.score;
  
  const totalSeconds = Math.floor(currentSession.total_time_seconds);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  document.getElementById('final-time').textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;
  
  // Check if it's a personal best (simplified - would need to fetch best scores)
  // For now, we'll skip this check
}

// Initialize
document.addEventListener('DOMContentLoaded', function() {
  // Difficulty selection
  document.querySelectorAll('.difficulty-card').forEach(card => {
    card.addEventListener('click', function() {
      const difficulty = this.dataset.difficulty;
      startRush(difficulty);
    });
  });
  
  // Start new rush button
  document.getElementById('start-new-rush')?.addEventListener('click', function() {
    currentSession = null;
    currentCard = null;
    stopTimer();
    showScreen('selection');
  });
  
  // If we have an active session, show it
  if (currentSession && !currentSession.completed_at) {
    showScreen('active');
    startTimer();
    updateStats();
    
    // Fetch current session and next flashcard
    fetch('/puzzle-rush/current')
      .then(r => {
        if (!r.ok) {
          throw new Error('Failed to fetch current session');
        }
        return r.json();
      })
      .then(data => {
        if (data && data.session && data.session.id) {
          // Update session data
          currentSession = data.session;
          updateStats();
          
          // Load flashcard if available
          if (data.card) {
            currentCard = data.card;
            loadFlashcard(data.card);
          } else {
            // No flashcard available - show message
            const container = document.getElementById('flashcard-container');
            container.innerHTML = `
              <div class="box">
                <p class="has-text-centered">No flashcards available. Please add some games first.</p>
                <div class="has-text-centered mt-3">
                  <a href="/flashcards" class="button is-primary">Go to Flashcards</a>
                </div>
              </div>
            `;
          }
        } else {
          // No active session found
          showScreen('selection');
        }
      })
      .catch(err => {
        console.error('Failed to get current session:', err);
        // If there's an error, check if we should show selection or keep active
        // For now, show selection to be safe
        showScreen('selection');
      });
  } else if (currentSession && currentSession.completed_at) {
    showResults();
  } else {
    showScreen('selection');
  }
});
