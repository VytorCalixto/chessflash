// Review form submission and quality calculation
export function calculateQuality(isCorrect, attemptCount, timeSeconds, showAnswerClicked, maxAttempts) {
  // If user clicked "Show answer", it's definitely "Again"
  if (showAnswerClicked) {
    return 0; // Again
  }
  
  // If wrong after max attempts, it's "Again"
  if (!isCorrect && attemptCount >= maxAttempts) {
    return 0; // Again
  }
  
  // If correct on first try
  if (isCorrect && attemptCount === 1) {
    if (timeSeconds < 10) {
      return 3; // Easy - fast and correct
    } else if (timeSeconds < 30) {
      return 2; // Good - correct but took some time
    } else {
      return 1; // Hard - correct but slow
    }
  }
  
  // If correct but took multiple attempts
  if (isCorrect && attemptCount > 1) {
    if (timeSeconds < 20) {
      return 2; // Good - eventually got it quickly
    } else {
      return 1; // Hard - took multiple tries and time
    }
  }
  
  // Default fallback (shouldn't reach here normally)
  return 1; // Hard
}

export function showReviewForm(reviewForm, cardStartTime, wasCorrect, attemptCount, showAnswerClicked, autoSubmit, maxAttempts) {
  // Calculate time taken
  const timeElapsed = cardStartTime ? (Date.now() - cardStartTime) / 1000 : 0;
  
  // Calculate quality automatically
  const quality = calculateQuality(wasCorrect, attemptCount, timeElapsed, showAnswerClicked, maxAttempts);
  
  // Quality labels for display
  const qualityLabels = {
    0: "Again",
    1: "Hard", 
    2: "Good",
    3: "Easy"
  };
  
  // Quality emojis
  const qualityEmoji = {
    0: '🔄',
    1: '⚠️',
    2: '✅',
    3: '⭐'
  };
  
  // Show feedback about auto-rating
  const ratingMessage = document.getElementById("auto-rating-message");
  if (ratingMessage) {
    ratingMessage.innerHTML = `
      <span class="has-text-weight-semibold">${qualityEmoji[quality]} ${qualityLabels[quality]}</span>
      <span class="has-text-grey is-size-7 ml-2">
        ${timeElapsed.toFixed(1)}s • ${attemptCount} attempt${attemptCount !== 1 ? 's' : ''}
      </span>
    `;
  }
  
  // Update rating badge styling
  const ratingBadge = document.getElementById('rating-badge');
  if (ratingBadge) {
    ratingBadge.className = `rating-badge ${qualityLabels[quality].toLowerCase()}`;
  }
  
  // Clear existing inputs and buttons (in case form is shown multiple times)
  // But preserve game_id and card_index if they exist (for game-filtered flashcards)
  const existingInputs = reviewForm.querySelectorAll('input[type="hidden"]');
  let gameIdValue = null;
  let cardIndexValue = null;
  
  // Preserve game_id and card_index values, remove only other inputs
  existingInputs.forEach(input => {
    if (input.name === 'game_id') {
      gameIdValue = input.value;
      // Keep this input, don't remove it
    } else if (input.name === 'card_index') {
      cardIndexValue = input.value;
      // Keep this input, don't remove it
    } else {
      // Remove only non-game inputs (time_seconds, attempt_count, quality, etc.)
      input.remove();
    }
  });
  
  // If game_id/card_index weren't in form, try to get them from URL
  if (!gameIdValue || !cardIndexValue) {
    const urlParams = new URLSearchParams(window.location.search);
    if (!gameIdValue) {
      gameIdValue = urlParams.get('game_id');
      if (gameIdValue) {
        // Add game_id input if we got it from URL
        const gameIdInput = document.createElement('input');
        gameIdInput.type = 'hidden';
        gameIdInput.name = 'game_id';
        gameIdInput.value = gameIdValue;
        reviewForm.appendChild(gameIdInput);
      }
    }
    if (!cardIndexValue) {
      cardIndexValue = urlParams.get('card_index');
      if (cardIndexValue) {
        // Add card_index input if we got it from URL
        const cardIndexInput = document.createElement('input');
        cardIndexInput.type = 'hidden';
        cardIndexInput.name = 'card_index';
        cardIndexInput.value = cardIndexValue;
        reviewForm.appendChild(cardIndexInput);
      }
    }
  }
  
  const nextCardContainer = document.getElementById('next-card-container');
  if (nextCardContainer) {
    nextCardContainer.innerHTML = '';
  }
  
  // Add hidden inputs
  const timeInput = document.createElement('input');
  timeInput.type = 'hidden';
  timeInput.name = 'time_seconds';
  timeInput.value = timeElapsed.toFixed(2);
  reviewForm.appendChild(timeInput);
  
  const attemptInput = document.createElement('input');
  attemptInput.type = 'hidden';
  attemptInput.name = 'attempt_count';
  attemptInput.value = attemptCount;
  reviewForm.appendChild(attemptInput);
  
  // Add hidden quality input
  const qualityInput = document.createElement('input');
  qualityInput.type = 'hidden';
  qualityInput.name = 'quality';
  qualityInput.value = quality;
  reviewForm.appendChild(qualityInput);
  
  // Show the form
  reviewForm.classList.remove("is-hidden");
  
  if (autoSubmit) {
    // Auto-submit after a short delay to show the feedback
    setTimeout(() => {
      reviewForm.submit();
    }, 1500); // Show feedback for 1.5 seconds before auto-submitting
  } else {
    // Add "Next card" button for manual submission
    const nextCardContainer = document.getElementById('next-card-container');
    if (nextCardContainer) {
      const nextButton = document.createElement('button');
      nextButton.type = 'submit';
      nextButton.className = 'button is-primary is-medium is-fullwidth';
      nextButton.innerHTML = `
        <span>Continue to Next Card</span>
        <span class="icon is-small">
          <i class="fas fa-arrow-right"></i>
        </span>
      `;
      nextButton.style.animation = 'fadeInUp 0.3s ease';
      nextCardContainer.appendChild(nextButton);
    }
  }
}
