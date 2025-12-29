// Chessground initialization and move handling
import { updateEvalBar, showEvalLoading, hideEvalLoading } from './eval-bar.js';

export function getLegalMoves(chess) {
  const dests = new Map();
  const squares = ['a1','a2','a3','a4','a5','a6','a7','a8',
                   'b1','b2','b3','b4','b5','b6','b7','b8',
                   'c1','c2','c3','c4','c5','c6','c7','c8',
                   'd1','d2','d3','d4','d5','d6','d7','d8',
                   'e1','e2','e3','e4','e5','e6','e7','e8',
                   'f1','f2','f3','f4','f5','f6','f7','f8',
                   'g1','g2','g3','g4','g5','g6','g7','g8',
                   'h1','h2','h3','h4','h5','h6','h7','h8'];
  squares.forEach(s => {
    const moves = chess.moves({ square: s, verbose: true });
    if (moves.length) {
      dests.set(s, moves.map(m => m.to));
    }
  });
  return dests;
}

export function moveToUci(moveObj) {
  if (!moveObj) return "";
  const promo = moveObj.promotion ? moveObj.promotion : "";
  return `${moveObj.from}${moveObj.to}${promo}`;
}

export function resetBoard(chess, initialFen, lastMove, sideToMove, cg, evalFill, evalLabel, evalBefore, mateBefore) {
  // Reset chess.js to initial position
  chess.load(initialFen);
  // Reset chessground board with initial state
  cg.set({ 
    fen: initialFen,
    lastMove: lastMove, // Restore the last move highlighting
    turnColor: sideToMove,
    movable: {
      free: false,
      color: sideToMove,
      dests: getLegalMoves(chess)
    }
  });
  // Clear any drawn shapes (arrows)
  cg.setShapes([]);
  // Reset eval bar to initial position
  updateEvalBar(evalFill, evalLabel, evalBefore, mateBefore);
}

export async function handleMove(
  orig, dest, chess, cg, bestMove, maxAttempts,
  attemptCount, setAttemptCount, isCompleted,
  evalFill, evalLabel, evalAfter, mateAfter,
  revealResult, attemptIndicator, attemptCountDisplay
) {
  if (isCompleted()) return;

  const move = chess.move({ from: orig, to: dest, promotion: 'q' });
  if (!move) {
    // Revert the move if invalid
    cg.set({ fen: chess.fen() });
    return;
  }

  // Update chessground FEN to match chess.js
  cg.set({ fen: chess.fen() });

  const played = moveToUci(move);
  const isCorrect = played === bestMove;
  const newAttemptCount = attemptCount + 1;
  setAttemptCount(newAttemptCount);
  
  // Update attempt indicator
  if (attemptIndicator && attemptCountDisplay) {
    attemptIndicator.classList.remove("is-hidden");
    attemptCountDisplay.textContent = `${newAttemptCount}/${maxAttempts}`;
  }
  
  // Show loading state on eval bar
  showEvalLoading(evalLabel);
  
  // Get evaluation for the user's actual move
  try {
    const response = await fetch(`/api/evaluate?fen=${encodeURIComponent(chess.fen())}`);
    if (response.ok) {
      const data = await response.json();
      const userEval = data.cp / 100; // Convert centipawns to pawns
      const userMate = data.mate;
      
      // Update eval bar with the user's move evaluation
      hideEvalLoading(evalLabel);
      updateEvalBar(evalFill, evalLabel, userEval, userMate);
      
      // If the move is correct, add visual feedback
      if (isCorrect) {
        setTimeout(() => {
          // Briefly highlight the eval bar in green
          evalLabel.style.transition = "background-color 0.3s";
          evalLabel.style.backgroundColor = "#d4edda";
          setTimeout(() => {
            evalLabel.style.backgroundColor = "";
          }, 500);
        }, 100);
      }
    } else {
      // Fallback: if evaluation fails, show the expected eval for correct moves
      hideEvalLoading(evalLabel);
      if (isCorrect) {
        updateEvalBar(evalFill, evalLabel, evalAfter, mateAfter);
      }
      console.error("Failed to evaluate position");
    }
  } catch (error) {
    console.error("Error evaluating position:", error);
    // Fallback behavior
    hideEvalLoading(evalLabel);
    if (isCorrect) {
      updateEvalBar(evalFill, evalLabel, evalAfter, mateAfter);
    }
  }
  
  if (isCorrect) {
    // Correct answer - show full feedback and complete
    revealResult(true, played, true);
  } else if (newAttemptCount >= maxAttempts) {
    // Max attempts reached - show full feedback and complete
    revealResult(false, played, true);
  } else {
    // Wrong answer but still have attempts left - show brief feedback
    revealResult(false, played, false);
    // Reset board after a short delay to allow user to see the move
    setTimeout(() => {
      // This will be handled by the main module
    }, 1500);
  }
  
  return { isCorrect, played, newAttemptCount };
}

export function initializeChessground(
  boardEl, fen, sideToMove, lastMove, chess,
  handleMoveCallback
) {
  // Check for Chessground - it might be exposed as window.Chessground or just Chessground
  const ChessgroundLib = window.Chessground || (typeof Chessground !== 'undefined' ? Chessground : null);
  if (typeof Chess === "undefined" || !ChessgroundLib) {
    throw new Error("Chess or Chessground not loaded");
  }
  
  return ChessgroundLib(boardEl, {
    fen: fen,
    orientation: sideToMove,
    turnColor: sideToMove,
    lastMove: lastMove,
    movable: {
      free: false,
      color: sideToMove,
      dests: getLegalMoves(chess),
      events: {
        after: handleMoveCallback
      }
    },
    drawable: {
      enabled: true,
      visible: true,
      brushes: {
        green: { key: 'g', color: '#15781B', opacity: 0.6, lineWidth: 10 },
        red: { key: 'r', color: '#882020', opacity: 0.6, lineWidth: 10 }
      }
    }
  });
}

export function setupPlayerNames(sideToMove, whitePlayer, blackPlayer) {
  const nameTop = document.getElementById("name-top");
  const nameBottom = document.getElementById("name-bottom");
  const iconTop = document.getElementById("icon-top");
  const iconBottom = document.getElementById("icon-bottom");
  
  if (sideToMove === "white") {
    // White at bottom, black at top
    nameBottom.textContent = whitePlayer;
    nameTop.textContent = blackPlayer;
    iconBottom.classList.add("white");
    iconTop.classList.add("black");
  } else {
    // Black at bottom, white at top
    nameBottom.textContent = blackPlayer;
    nameTop.textContent = whitePlayer;
    iconBottom.classList.add("black");
    iconTop.classList.add("white");
  }
}
