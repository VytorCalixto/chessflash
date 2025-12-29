// Main entry point for flashcard functionality
import { updateEvalBar } from './eval-bar.js';
import { showReviewForm } from './review.js';
import { 
  initializeChessground, 
  resetBoard, 
  handleMove as handleMoveBoard,
  setupPlayerNames 
} from './board.js';

function initBoard() {
  // Get data from JSON script block
  const dataScript = document.getElementById('flashcard-data');
  if (!dataScript) {
    console.error('Flashcard data not found');
    return;
  }
  
  let cardData;
  try {
    cardData = JSON.parse(dataScript.textContent);
  } catch (e) {
    throw e;
  }
  
  // Defensive parsing: ensure FEN is a clean string (remove any extra quotes)
  let fen = cardData.fen;
  if (typeof fen === 'string') {
    // Remove surrounding quotes if present (handles double-encoding)
    fen = fen.replace(/^["']+|["']+$/g, '');
    // Trim whitespace
    fen = fen.trim();
  }
  
  // Validate FEN format (should have at least 4 space-separated parts)
  if (!fen || typeof fen !== 'string' || fen.split(' ').length < 4) {
    console.error('Invalid FEN format:', fen);
    return;
  }
  
  const bestMove = cardData.bestMove.trim();
  const mateBefore = cardData.mateBefore;
  const mateAfter = cardData.mateAfter;
  const evalBefore = parseFloat(cardData.evalBefore) / 100;
  const evalAfter = parseFloat(cardData.evalAfter) / 100;
  const evalDiff = parseFloat(cardData.evalDiff) / 100;
  const classification = cardData.classification.toLowerCase();
  const movePlayed = cardData.movePlayed;
  const prevMovePlayed = cardData.prevMovePlayed;
  const whitePlayer = cardData.whitePlayer;
  const blackPlayer = cardData.blackPlayer;
  
  // Parse the last move for highlighting (opponent's move that led to this position)
  let lastMove = null;
  if (prevMovePlayed && prevMovePlayed.length >= 4) {
    const from = prevMovePlayed.substring(0, 2);
    const to = prevMovePlayed.substring(2, 4);
    lastMove = [from, to];
  }

  const promptEl = document.getElementById("prompt-text");
  const feedbackEl = document.getElementById("feedback-text");
  const lossEl = document.getElementById("loss-text");
  const metaEl = document.getElementById("meta-text");
  const reviewForm = document.getElementById("review-form");
  const showAnswerBtn = document.getElementById("show-answer");
  const evalFill = document.getElementById("eval-fill");
  const evalLabel = document.getElementById("eval-label");
  const feedbackBox = document.getElementById("feedback-box");
  const attemptIndicator = document.getElementById("attempt-indicator");
  const attemptCountDisplay = document.getElementById("attempt-count-display");

  let chess;
  try {
    chess = new Chess(fen);
    const loadedFen = chess.fen();
    const isValid = loadedFen === fen || loadedFen.split(' ')[0] === fen.split(' ')[0];
    
    // Check if Chess.js actually loaded the FEN correctly
    if (!isValid) {
      console.error('Chess.js failed to load FEN correctly. Expected:', fen, 'Got:', loadedFen);
      // Try to reload with the FEN directly
      try {
        chess.load(fen);
      } catch (reloadErr) {
        console.error('Failed to reload FEN:', reloadErr);
        return;
      }
    }
  } catch (e) {
    console.error('Failed to initialize Chess.js:', e);
    return;
  }
  
  const initialFen = fen; // Store initial position for resetting after wrong moves
  const sideToMove = fen.split(" ")[1] === "w" ? "white" : "black";
  const maxAttempts = 3;
  
  let attemptCount = 0;
  let isCompleted = false; // Track if card is completed (correct answer or show answer clicked)
  let cardStartTime = Date.now(); // Track when card is displayed
  let wasCorrect = false; // Track if the answer was correct
  let showAnswerClicked = false; // Track if user clicked "Show answer"
  let cg = null;

  function setAttemptCount(count) {
    attemptCount = count;
  }

  function getIsCompleted() {
    return isCompleted;
  }

  function contextualPrompt() {
    // Determine if user is playing as white or black
    const isUserWhite = sideToMove === "white";
    
    // Convert evaluation to user's perspective
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

  function classificationNote() {
    switch (classification) {
      case "blunder": return "You missed a critical idea here.";
      case "mistake": return "There was a better option.";
      case "inaccuracy": return "A small improvement was possible.";
      default: return "";
    }
  }

  function revealResult(isCorrect, moveUci, showFullFeedback) {
    const loss = Math.abs(evalDiff).toFixed(1);
    
    // Track correctness
    wasCorrect = isCorrect;
    
    if (isCorrect) {
      // Correct answer
      isCompleted = true;
      feedbackEl.textContent = "Excellent! You found the best move.";
      feedbackEl.classList.remove("has-text-danger", "has-text-info");
      feedbackEl.classList.add("has-text-success");
      lossEl.textContent = "";
      metaEl.textContent = classificationNote();
      
      // Update feedback box styling
      if (feedbackBox) {
        feedbackBox.classList.remove("error", "info");
        feedbackBox.classList.add("success");
      }
      
      // Auto-submit review (user got it right, no need to review)
      showReviewForm(reviewForm, cardStartTime, wasCorrect, attemptCount, showAnswerClicked, true, maxAttempts);
      cg.set({ movable: { color: undefined } });
    } else if (showFullFeedback) {
      // Max attempts reached - show full feedback
      isCompleted = true;
      feedbackEl.textContent = `Not quite. The best move was ${bestMove}.`;
      feedbackEl.classList.remove("has-text-success", "has-text-info");
      feedbackEl.classList.add("has-text-danger");
      
      // Update feedback box styling
      if (feedbackBox) {
        feedbackBox.classList.remove("success", "info");
        feedbackBox.classList.add("error");
      }
      
      // Draw arrow for best move
      if (bestMove && bestMove.length >= 4) {
        const from = bestMove.substring(0, 2);
        const to = bestMove.substring(2, 4);
        cg.setShapes([{ orig: from, dest: to, brush: 'green' }]);
      }
      
      if (mateAfter !== null && mateAfter !== undefined) {
        lossEl.textContent = mateAfter > 0
          ? "Your move allows mate for White."
          : "Your move allows mate for Black.";
      } else {
        lossEl.textContent = `Your original move lost ${loss} pawns of advantage.`;
      }
      metaEl.textContent = classificationNote();
      
      // Show "Next card" button (user failed, let them review)
      showReviewForm(reviewForm, cardStartTime, wasCorrect, attemptCount, showAnswerClicked, false, maxAttempts);
      cg.set({ movable: { color: undefined } });
    } else {
      // Brief feedback for wrong moves (still have attempts left)
      const attemptsRemaining = maxAttempts - attemptCount;
      feedbackEl.textContent = `Not quite. Try again (${attemptsRemaining} ${attemptsRemaining === 1 ? 'attempt' : 'attempts'} remaining).`;
      feedbackEl.classList.remove("has-text-success", "has-text-info");
      feedbackEl.classList.add("has-text-danger");
      lossEl.textContent = "";
      metaEl.textContent = "";
    }
  }

  async function handleMoveCallback(orig, dest) {
    const result = await handleMoveBoard(
      orig, dest, chess, cg, bestMove, maxAttempts,
      attemptCount, setAttemptCount, getIsCompleted,
      evalFill, evalLabel, evalAfter, mateAfter,
      revealResult, attemptIndicator, attemptCountDisplay
    );
    
    if (result && !result.isCorrect && result.newAttemptCount < maxAttempts) {
      // Reset board after a short delay
      setTimeout(() => {
        resetBoard(chess, initialFen, lastMove, sideToMove, cg, evalFill, evalLabel, evalBefore, mateBefore);
      }, 1500);
    }
  }

  try {
    const boardEl = document.getElementById('board');
    cg = initializeChessground(boardEl, fen, sideToMove, lastMove, chess, handleMoveCallback);
  } catch (err) {
    console.error(err);
    return;
  }

  showAnswerBtn.addEventListener("click", function() {
    if (isCompleted) return;
    
    showAnswerClicked = true;
    wasCorrect = false;
    isCompleted = true;
    
    // Draw arrow for best move
    if (bestMove && bestMove.length >= 4) {
      const from = bestMove.substring(0, 2);
      const to = bestMove.substring(2, 4);
      cg.setShapes([{ orig: from, dest: to, brush: 'green' }]);
    }
    
    feedbackEl.textContent = `Best move: ${bestMove}`;
    feedbackEl.classList.remove("has-text-danger", "has-text-success");
    feedbackEl.classList.add("has-text-info");
    metaEl.textContent = classificationNote();
    lossEl.textContent = `Original move (${movePlayed}) lost ${Math.abs(evalDiff).toFixed(1)} pawns.`;
    
    // Update feedback box styling
    if (feedbackBox) {
      feedbackBox.classList.remove("success", "error");
      feedbackBox.classList.add("info");
    }
    
    // Show "Next card" button (user revealed answer, let them review)
    showReviewForm(reviewForm, cardStartTime, wasCorrect, attemptCount, showAnswerClicked, false, maxAttempts);
    cg.set({ movable: { color: undefined } });
  });

  // Initial UI state
  promptEl.textContent = contextualPrompt();
  // Initialize eval bar immediately to avoid showing +0.0
  updateEvalBar(evalFill, evalLabel, evalBefore, mateBefore);
  setupPlayerNames(sideToMove, whitePlayer, blackPlayer);
}

// Wait for window load to ensure all scripts are loaded
if (window.addEventListener) {
  window.addEventListener('load', initBoard);
} else if (window.attachEvent) {
  window.attachEvent('onload', initBoard);
} else {
  // Fallback: try after a delay
  setTimeout(initBoard, 500);
}
