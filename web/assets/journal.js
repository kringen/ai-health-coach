// Journal entry functionality
class JournalEntry {
    constructor() {
        this.currentDate = this.getDateFromURL() || new Date();
        this.init();
    }

    init() {
        this.setupDate();
        this.bindEvents();
        this.loadEntry();
    }

    getDateFromURL() {
        const params = new URLSearchParams(window.location.search);
        const dateParam = params.get('date');
        
        if (dateParam && dateParam !== 'today') {
            return new Date(dateParam);
        }
        
        return new Date(); // Default to today
    }

    setupDate() {
        const dateElement = document.getElementById('entryDate');
        if (dateElement) {
            dateElement.textContent = this.formatDisplayDate(this.currentDate);
        }
    }

    bindEvents() {
        const form = document.getElementById('journalForm');
        if (form) {
            form.addEventListener('submit', (e) => this.handleSubmit(e));
        }

        const evaluateDaily = document.getElementById('evaluateDaily');
        if (evaluateDaily) {
            evaluateDaily.addEventListener('click', () => this.evaluateEntry('daily'));
        }

        const evaluateHistorical = document.getElementById('evaluateHistorical');
        if (evaluateHistorical) {
            evaluateHistorical.addEventListener('click', () => this.evaluateEntry('historical'));
        }
    }

    async loadEntry() {
        try {
            const dateStr = this.formatDate(this.currentDate);
            const response = await fetch(`/api/journal/${dateStr}`, {
                headers: this.getAuthHeaders()
            });

            if (response.ok) {
                const entry = await response.json();
                this.populateForm(entry);
            } else if (response.status !== 404) {
                throw new Error('Failed to load journal entry');
            }
        } catch (error) {
            console.error('Error loading journal entry:', error);
            this.showMessage('Failed to load journal entry', 'error');
        }
    }

    populateForm(entry) {
        // Populate form fields with existing data
        const fields = [
            'foodIntake', 'waterIntake', 'bedTime', 'wakeTime', 
            'alcoholUnits', 'exerciseType', 'exerciseDuration', 'notes'
        ];

        fields.forEach(field => {
            const element = document.getElementById(field);
            if (element && entry[this.camelToSnake(field)] !== undefined) {
                element.value = entry[this.camelToSnake(field)] || '';
            }
        });

        // Handle radio buttons for sleep quality
        if (entry.sleep_quality) {
            const sleepRadio = document.getElementById(`sleep${entry.sleep_quality}`);
            if (sleepRadio) {
                sleepRadio.checked = true;
            }
        }

        // Handle radio buttons for exercise intensity
        if (entry.exercise_intensity) {
            const intensityRadio = document.getElementById(`intensity${entry.exercise_intensity}`);
            if (intensityRadio) {
                intensityRadio.checked = true;
            }
        }
    }

    async handleSubmit(e) {
        e.preventDefault();
        
        this.showLoading(true);
        
        try {
            const formData = this.collectFormData();
            const response = await fetch('/api/journal', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...this.getAuthHeaders()
                },
                body: JSON.stringify(formData)
            });

            if (response.ok) {
                this.showMessage('Journal entry saved successfully!', 'success');
            } else {
                throw new Error('Failed to save journal entry');
            }
        } catch (error) {
            console.error('Error saving journal entry:', error);
            this.showMessage('Failed to save journal entry', 'error');
        } finally {
            this.showLoading(false);
        }
    }

    collectFormData() {
        const data = {
            date: this.formatDate(this.currentDate),
            food_intake: document.getElementById('foodIntake').value || '',
            water_intake: parseFloat(document.getElementById('waterIntake').value) || 0,
            bed_time: document.getElementById('bedTime').value || null,
            wake_time: document.getElementById('wakeTime').value || null,
            alcohol_units: parseFloat(document.getElementById('alcoholUnits').value) || 0,
            exercise_type: document.getElementById('exerciseType').value || '',
            exercise_duration: parseInt(document.getElementById('exerciseDuration').value) || 0,
            notes: document.getElementById('notes').value || ''
        };

        // Get sleep quality from radio buttons
        const sleepQuality = document.querySelector('input[name="sleepQuality"]:checked');
        if (sleepQuality) {
            data.sleep_quality = parseInt(sleepQuality.value);
        }

        // Get exercise intensity from radio buttons
        const exerciseIntensity = document.querySelector('input[name="exerciseIntensity"]:checked');
        if (exerciseIntensity) {
            data.exercise_intensity = parseInt(exerciseIntensity.value);
        }

        return data;
    }

    async evaluateEntry(type) {
        try {
            // First save the current entry
            await this.handleSubmit(new Event('submit'));

            // Then navigate to chat page with evaluation type
            const dateStr = this.formatDate(this.currentDate);
            window.location.href = `/chat?date=${dateStr}&type=${type}`;
        } catch (error) {
            console.error('Error preparing evaluation:', error);
            this.showMessage('Please save your entry first', 'error');
        }
    }

    showLoading(show) {
        const loading = document.getElementById('loading');
        if (loading) {
            loading.style.display = show ? 'flex' : 'none';
        }
    }

    showMessage(text, type) {
        const messageEl = document.getElementById('message');
        if (messageEl) {
            messageEl.textContent = text;
            messageEl.className = `message ${type}`;
            messageEl.style.display = 'block';
            
            setTimeout(() => {
                messageEl.style.display = 'none';
            }, 5000);
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

    camelToSnake(str) {
        return str.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);
    }
}

// Initialize journal when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new JournalEntry();
});
