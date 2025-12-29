// Evaluation bar display logic
export function clampEval(val) {
  if (val > 10) return 10;
  if (val < -10) return -10;
  return val;
}

export function updateEvalBar(evalFill, evalLabel, value, mate) {
  if (mate !== null && mate !== undefined) {
    const isWhiteMate = mate > 0;
    const mateDisplay = Math.abs(mate);
    evalFill.style.height = "100%";
    evalLabel.textContent = (isWhiteMate ? "M" : "-M") + mateDisplay;
    if (isWhiteMate) {
      evalLabel.classList.remove("black-advantage");
      evalLabel.classList.add("white-advantage");
    } else {
      evalLabel.classList.remove("white-advantage");
      evalLabel.classList.add("black-advantage");
    }
    return;
  }
  // Ensure value is a number
  const numValue = typeof value === 'string' ? parseFloat(value) : value;
  // Eval is always from white's perspective (white at top, black at bottom in eval bar)
  const displayValue = numValue;
  const clamped = clampEval(displayValue);
  
  // Reset eval bar to normal orientation (white at top, black at bottom)
  const evalBar = evalFill.parentElement.parentElement;
  evalBar.style.flexDirection = "column";
  evalFill.style.top = "auto";
  evalFill.style.bottom = "0";
  
  const percent = ((clamped + 10) / 20) * 100;
  evalFill.style.height = percent + "%";
  evalLabel.textContent = (displayValue >= 0 ? "+" : "") + displayValue.toFixed(1);
  
  // Update label styling based on advantage (from white's perspective)
  if (displayValue >= 0) {
    evalLabel.classList.remove("black-advantage");
    evalLabel.classList.add("white-advantage");
  } else {
    evalLabel.classList.remove("white-advantage");
    evalLabel.classList.add("black-advantage");
  }
}

export function showEvalLoading(evalLabel) {
  evalLabel.textContent = "...";
  evalLabel.style.opacity = "0.6";
}

export function hideEvalLoading(evalLabel) {
  evalLabel.style.opacity = "1";
}
