// AI Chat functionality
class HealthChat {
    constructor() {
        this.currentDate = this.getDateFromURL();
        this.evaluationType = this.getEvaluationTypeFromURL() || 'daily';
        this.init();
    }

    init() {
        this.setupDate();
        this.bindEvents();
        this.loadHistory();
    }

    getDateFromURL() {
        const params = new URLSearchParams(window.location.search);
        const dateParam = params.get('date');
        
        if (dateParam) {
            // Parse date string carefully to avoid timezone issues
            const [year, month, day] = dateParam.split('-').map(Number);
            return new Date(year, month - 1, day); // month is 0-indexed
        }
        
        return new Date();
    }

    getEvaluationTypeFromURL() {
        const params = new URLSearchParams(window.location.search);
        return params.get('type') || 'daily';
    }

    setupDate() {
        const dateElement = document.getElementById('chatDate');
        if (dateElement) {
            dateElement.textContent = this.formatDisplayDate(this.currentDate);
        }
        
        // Set the evaluation type radio button
        const typeRadio = document.querySelector(`input[name="evaluationType"][value="${this.evaluationType}"]`);
        if (typeRadio) {
            typeRadio.checked = true;
        }
    }

    bindEvents() {
        const evaluateBtn = document.getElementById('requestEvaluation');
        if (evaluateBtn) {
            evaluateBtn.addEventListener('click', () => this.requestEvaluation());
        }

        // Follow-up chat functionality
        const sendFollowupBtn = document.getElementById('sendFollowup');
        if (sendFollowupBtn) {
            sendFollowupBtn.addEventListener('click', () => this.sendFollowupQuestion());
        }

        const followupInput = document.getElementById('followupInput');
        if (followupInput) {
            followupInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') {
                    this.sendFollowupQuestion();
                }
            });
        }

        // Question pill buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('pill-btn')) {
                const question = e.target.getAttribute('data-question');
                this.askPredefinedQuestion(question);
            }
        });
    }

    async loadHistory() {
        try {
            const dateStr = this.formatDate(this.currentDate);
            const response = await fetch(`/api/journal/${dateStr}/chat`, {
                headers: this.getAuthHeaders()
            });

            if (!response.ok) {
                return;
            }

            const data = await response.json();
            const messages = data.messages || [];

            if (messages.length === 0) {
                return;
            }

            const messagesContainer = document.getElementById('chatMessages');
            messages.forEach((message) => this.renderStoredMessage(messagesContainer, message));

            this.showFollowupOptions();
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        } catch (error) {
            console.error('Error loading chat history:', error);
        }
    }

    renderStoredMessage(messagesContainer, message) {
        if (message.role === 'user') {
            messagesContainer.appendChild(this.createMessage('user', message.content));
            return;
        }

        if (message.format === 'markdown') {
            messagesContainer.appendChild(this.createMarkdownMessage(message.content));
        } else if (message.format === 'json') {
            try {
                const evaluation = JSON.parse(message.content);
                messagesContainer.appendChild(this.createEvaluationMessage(evaluation));
            } catch (error) {
                messagesContainer.appendChild(this.createMessage('ai', message.content));
            }
        } else {
            messagesContainer.appendChild(this.createMessage('ai', message.content));
        }
    }

    async requestEvaluation() {
        const selectedType = document.querySelector('input[name="evaluationType"]:checked');
        const evaluationType = selectedType ? selectedType.value : 'daily';
        
        this.showLoading(true);
        
        try {
            const dateStr = this.formatDate(this.currentDate);
            const response = await fetch('/api/ollama/evaluate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...this.getAuthHeaders()
                },
                body: JSON.stringify({
                    date: dateStr,
                    evaluation_type: evaluationType
                })
            });

            if (response.ok) {
                const contentType = response.headers.get('content-type');
                if (contentType && contentType.includes('text/markdown')) {
                    // Handle markdown response
                    const markdownContent = await response.text();
                    this.displayMarkdownEvaluation(markdownContent, evaluationType);
                } else {
                    // Handle JSON response (fallback)
                    const evaluation = await response.json();
                    this.displayEvaluation(evaluation, evaluationType);
                }
            } else {
                const error = await response.text();
                throw new Error(error || 'Failed to get evaluation');
            }
        } catch (error) {
            console.error('Error getting evaluation:', error);
            this.displayError(error.message);
        } finally {
            this.showLoading(false);
        }
    }

    displayEvaluation(evaluation, type) {
        const messagesContainer = document.getElementById('chatMessages');
        
        // Add user message
        const userMessage = this.createMessage('user', `Please provide a ${type} health evaluation for ${this.formatDisplayDate(this.currentDate)}.`);
        messagesContainer.appendChild(userMessage);

        // Add AI response
        const aiMessage = this.createEvaluationMessage(evaluation);
        messagesContainer.appendChild(aiMessage);

        // Show follow-up options after evaluation
        this.showFollowupOptions();

        // Scroll to bottom
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    displayMarkdownEvaluation(markdownContent, type) {
        const messagesContainer = document.getElementById('chatMessages');
        
        // Add user message
        const userMessage = this.createMessage('user', `Please provide a ${type} health evaluation for ${this.formatDisplayDate(this.currentDate)}.`);
        messagesContainer.appendChild(userMessage);

        // Add AI response with markdown content
        const aiMessage = this.createMarkdownMessage(markdownContent);
        messagesContainer.appendChild(aiMessage);

        // Show follow-up options after evaluation
        this.showFollowupOptions();

        // Scroll to bottom
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    createMessage(sender, content) {
        const messageDiv = document.createElement('div');
        messageDiv.className = `message ${sender}-message`;

        const avatar = document.createElement('div');
        avatar.className = 'message-avatar';
        avatar.textContent = sender === 'user' ? '👤' : '🤖';

        const contentDiv = document.createElement('div');
        contentDiv.className = 'message-content';
        
        if (typeof content === 'string') {
            contentDiv.innerHTML = this.formatText(content);
        } else {
            contentDiv.appendChild(content);
        }

        messageDiv.appendChild(avatar);
        messageDiv.appendChild(contentDiv);

        return messageDiv;
    }

    createEvaluationMessage(evaluation) {
        const contentDiv = document.createElement('div');
        
        // Health Score
        const scoreDiv = document.createElement('div');
        scoreDiv.className = 'evaluation-score';
        scoreDiv.innerHTML = `
            <h4>Health Score</h4>
            <div class="score-display">
                <span class="score-number">${evaluation.score}</span>
                <span class="score-total">/10</span>
            </div>
        `;
        scoreDiv.style.cssText = `
            background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
            padding: 1rem;
            border-radius: 0.5rem;
            margin-bottom: 1rem;
            text-align: center;
        `;
        contentDiv.appendChild(scoreDiv);

        // Evaluation Text
        const evaluationDiv = document.createElement('div');
        evaluationDiv.className = 'evaluation-text';
        evaluationDiv.innerHTML = `
            <h4>Analysis</h4>
            <p>${this.formatText(evaluation.evaluation)}</p>
        `;
        evaluationDiv.style.cssText = `
            margin-bottom: 1rem;
        `;
        contentDiv.appendChild(evaluationDiv);

        // Suggestions
        if (evaluation.suggestions && evaluation.suggestions.length > 0) {
            const suggestionsDiv = document.createElement('div');
            suggestionsDiv.className = 'evaluation-suggestions';
            suggestionsDiv.innerHTML = `
                <h4>Suggestions</h4>
                <ul>
                    ${evaluation.suggestions.map(suggestion => `<li>${this.formatText(suggestion)}</li>`).join('')}
                </ul>
            `;
            suggestionsDiv.style.cssText = `
                background: #f0fdf4;
                padding: 1rem;
                border-radius: 0.5rem;
                border-left: 4px solid #059669;
            `;
            contentDiv.appendChild(suggestionsDiv);
        }

        return this.createMessage('ai', contentDiv);
    }

    createMarkdownMessage(markdownContent) {
        const contentDiv = document.createElement('div');
        contentDiv.className = 'markdown-content';
        
        // Extract health score from markdown if present
        const scoreMatch = markdownContent.match(/(?:Health Score|Score):\s*(\d+(?:\.\d+)?)\s*(?:\/\s*10)?/i);
        if (scoreMatch) {
            const score = parseFloat(scoreMatch[1]);
            const scoreDiv = document.createElement('div');
            scoreDiv.className = 'evaluation-score';
            scoreDiv.innerHTML = `
                <h4>Health Score</h4>
                <div class="score-display">
                    <span class="score-number">${score}</span>
                    <span class="score-total">/10</span>
                </div>
            `;
            scoreDiv.style.cssText = `
                background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
                padding: 1rem;
                border-radius: 0.5rem;
                margin-bottom: 1rem;
                text-align: center;
            `;
            contentDiv.appendChild(scoreDiv);
        }

        // Format the markdown content
        const markdownDiv = document.createElement('div');
        markdownDiv.className = 'markdown-evaluation';
        markdownDiv.innerHTML = this.formatMarkdown(markdownContent);
        markdownDiv.style.cssText = `
            line-height: 1.6;
            color: #374151;
        `;
        contentDiv.appendChild(markdownDiv);

        return this.createMessage('ai', contentDiv);
    }

    formatMarkdown(markdown) {
        return markdown
            // Headers
            .replace(/^### (.*$)/gim, '<h3 style="color: #1f2937; margin: 1.5rem 0 0.5rem 0;">$1</h3>')
            .replace(/^## (.*$)/gim, '<h2 style="color: #1f2937; margin: 1.5rem 0 0.5rem 0;">$1</h2>')
            .replace(/^# (.*$)/gim, '<h1 style="color: #1f2937; margin: 1.5rem 0 0.5rem 0;">$1</h1>')
            // Bold and italic
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            .replace(/\*(.*?)\*/g, '<em>$1</em>')
            // Lists
            .replace(/^\- (.*$)/gim, '<li style="margin: 0.25rem 0;">$1</li>')
            .replace(/(<li.*<\/li>)/s, '<ul style="margin: 0.5rem 0; padding-left: 1.5rem;">$1</ul>')
            // Line breaks
            .replace(/\n\n/g, '</p><p style="margin: 1rem 0;">')
            .replace(/\n/g, '<br>')
            // Wrap in paragraphs
            .replace(/^(.+)$/gm, '<p style="margin: 1rem 0;">$1</p>')
            // Clean up multiple paragraph tags
            .replace(/<p[^>]*><\/p>/g, '')
            .replace(/(<p[^>]*>)\s*(<h[1-6])/g, '$2')
            .replace(/(<\/h[1-6]>)\s*(<\/p>)/g, '$1');
    }

    displayError(message) {
        const messagesContainer = document.getElementById('chatMessages');
        
        const errorContent = document.createElement('div');
        errorContent.innerHTML = `
            <div class="error-message">
                <h4>⚠️ Evaluation Error</h4>
                <p>${message}</p>
                <p><small>The AI evaluation service may be temporarily unavailable. Please try again later.</small></p>
            </div>
        `;
        errorContent.style.cssText = `
            background: #fef2f2;
            color: #dc2626;
            padding: 1rem;
            border-radius: 0.5rem;
            border-left: 4px solid #dc2626;
        `;

        const errorMessage = this.createMessage('ai', errorContent);
        messagesContainer.appendChild(errorMessage);

        // Scroll to bottom
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    showFollowupOptions() {
        const followupContainer = document.getElementById('followupContainer');
        if (followupContainer) {
            followupContainer.style.display = 'block';
        }
    }

    sendFollowupQuestion() {
        const input = document.getElementById('followupInput');
        const question = input.value.trim();
        
        if (!question) return;

        this.askQuestion(question);
        input.value = '';
    }

    askPredefinedQuestion(question) {
        this.askQuestion(question);
    }

    async askQuestion(question) {
        const messagesContainer = document.getElementById('chatMessages');
        
        // Add user message
        const userMessage = this.createMessage('user', question);
        messagesContainer.appendChild(userMessage);
        
        // Scroll to bottom
        messagesContainer.scrollTop = messagesContainer.scrollHeight;

        this.showLoading(true);

        try {
            const dateStr = this.formatDate(this.currentDate);
            const response = await fetch('/api/ollama/evaluate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...this.getAuthHeaders()
                },
                body: JSON.stringify({
                    date: dateStr,
                    evaluation_type: 'followup',
                    question: question
                })
            });

            if (response.ok) {
                const contentType = response.headers.get('content-type');
                if (contentType && contentType.includes('text/markdown')) {
                    const markdownContent = await response.text();
                    const aiMessage = this.createMarkdownMessage(markdownContent);
                    messagesContainer.appendChild(aiMessage);
                } else {
                    const evaluation = await response.json();
                    const aiMessage = this.createEvaluationMessage(evaluation);
                    messagesContainer.appendChild(aiMessage);
                }
            } else {
                const error = await response.text();
                throw new Error(error || 'Failed to get response');
            }
        } catch (error) {
            console.error('Error getting answer:', error);
            this.displayError(error.message);
        } finally {
            this.showLoading(false);
            // Scroll to bottom
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }
    }

    formatText(text) {
        // Simple formatting for better readability
        return text
            .replace(/\n/g, '<br>')
            .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
            .replace(/\*(.*?)\*/g, '<em>$1</em>');
    }

    showLoading(show) {
        const loading = document.getElementById('evaluationLoading');
        if (loading) {
            loading.style.display = show ? 'flex' : 'none';
        }

        const button = document.getElementById('requestEvaluation');
        if (button) {
            button.disabled = show;
            button.textContent = show ? 'Analyzing...' : 'Get Health Evaluation';
        }
    }

    getAuthHeaders() {
        const token = localStorage.getItem('authToken');
        if (token) {
            return { 'Authorization': `Bearer ${token}` };
        }
        
        // For development mode
        return { 'X-Development-Mode': 'true' };
    }

    formatDate(date) {
        return date.toISOString().split('T')[0];
    }

    formatDisplayDate(date) {
        return date.toLocaleDateString('en-US', {
            weekday: 'long',
            year: 'numeric',
            month: 'long',
            day: 'numeric'
        });
    }
}

// Initialize chat when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new HealthChat();
});
