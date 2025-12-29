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
  
  // Clear any existing inputs and buttons (in case form is shown multiple times)
  const existingInputs = reviewForm.querySelectorAll('input[type="hidden"]');
  existingInputs.forEach(input => input.remove());
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
