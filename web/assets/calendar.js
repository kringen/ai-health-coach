// Calendar functionality
class HealthCalendar {
    constructor() {
        this.currentDate = new Date();
        this.selectedDate = null;
        this.entries = new Map(); // Store journal entries by date
        this.init();
    }

    init() {
        this.bindEvents();
        this.render();
        this.loadEntries();
    }

    bindEvents() {
        document.getElementById('prevMonth').addEventListener('click', () => {
            this.currentDate.setMonth(this.currentDate.getMonth() - 1);
            this.render();
        });

        document.getElementById('nextMonth').addEventListener('click', () => {
            this.currentDate.setMonth(this.currentDate.getMonth() + 1);
            this.render();
        });
    }

    render() {
        this.renderHeader();
        this.renderCalendar();
    }

    renderHeader() {
        const monthNames = [
            'January', 'February', 'March', 'April', 'May', 'June',
            'July', 'August', 'September', 'October', 'November', 'December'
        ];
        
        const monthYear = `${monthNames[this.currentDate.getMonth()]} ${this.currentDate.getFullYear()}`;
        document.getElementById('currentMonth').textContent = monthYear;
    }

    renderCalendar() {
        const calendar = document.getElementById('calendar');
        calendar.innerHTML = '';

        // Add day headers
        const dayHeaders = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
        dayHeaders.forEach(day => {
            const header = document.createElement('div');
            header.className = 'calendar-day-header';
            header.textContent = day;
            header.style.cssText = `
                background: #f8fafc;
                padding: 1rem;
                text-align: center;
                font-weight: 600;
                color: #64748b;
                font-size: 0.875rem;
            `;
            calendar.appendChild(header);
        });

        // Get first day of month and number of days
        const firstDay = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 1);
        const lastDay = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, 0);
        const startDate = new Date(firstDay);
        startDate.setDate(startDate.getDate() - firstDay.getDay());

        // Generate calendar days
        for (let i = 0; i < 42; i++) {
            const date = new Date(startDate);
            date.setDate(startDate.getDate() + i);
            
            const dayElement = this.createDayElement(date, firstDay, lastDay);
            calendar.appendChild(dayElement);
        }
    }

    createDayElement(date, firstDay, lastDay) {
        const dayDiv = document.createElement('div');
        dayDiv.className = 'calendar-day';
        
        const isCurrentMonth = date >= firstDay && date <= lastDay;
        const isToday = this.isSameDay(date, new Date());
        const dateKey = this.formatDate(date);
        const hasEntry = this.entries.has(dateKey);

        if (!isCurrentMonth) {
            dayDiv.classList.add('other-month');
        }
        
        if (isToday) {
            dayDiv.classList.add('today');
            dayDiv.style.background = '#dbeafe';
        }

        if (hasEntry) {
            dayDiv.classList.add('has-entry');
        }

        // Day number
        const dayNumber = document.createElement('div');
        dayNumber.className = 'day-number';
        dayNumber.textContent = date.getDate();
        dayDiv.appendChild(dayNumber);

        // Indicators for different health metrics
        if (hasEntry) {
            const indicators = this.createIndicators(this.entries.get(dateKey));
            dayDiv.appendChild(indicators);
        }

        // Click handler
        dayDiv.addEventListener('click', () => {
            this.selectDate(date);
        });

        return dayDiv;
    }

    createIndicators(entry) {
        const container = document.createElement('div');
        container.className = 'day-indicators';

        if (entry.water_intake > 0) {
            const indicator = document.createElement('div');
            indicator.className = 'indicator water';
            indicator.title = `Water: ${entry.water_intake}L`;
            container.appendChild(indicator);
        }

        if (entry.exercise_type) {
            const indicator = document.createElement('div');
            indicator.className = 'indicator exercise';
            indicator.title = `Exercise: ${entry.exercise_type}`;
            container.appendChild(indicator);
        }

        if (entry.sleep_quality) {
            const indicator = document.createElement('div');
            indicator.className = 'indicator sleep';
            indicator.title = `Sleep: ${entry.sleep_quality}/5`;
            container.appendChild(indicator);
        }

        return container;
    }

    selectDate(date) {
        // Remove previous selection
        document.querySelectorAll('.calendar-day.selected').forEach(el => {
            el.classList.remove('selected');
        });

        // Find and select the clicked day
        const dayElements = document.querySelectorAll('.calendar-day');
        dayElements.forEach(el => {
            const dayNumber = parseInt(el.querySelector('.day-number').textContent);
            if (dayNumber === date.getDate()) {
                el.classList.add('selected');
            }
        });

        this.selectedDate = date;
        
        // Navigate to journal page for selected date
        const dateStr = this.formatDate(date);
        window.location.href = `/journal?date=${dateStr}`;
    }

    async loadEntries() {
        try {
            // Load entries for current month
            const startDate = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 1);
            const endDate = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, 0);
            
            // For now, we'll load entries as we navigate to journal pages
            // In a real implementation, you might want to load a month's worth of data
            this.render(); // Re-render to show indicators
        } catch (error) {
            console.error('Failed to load calendar entries:', error);
        }
    }

    formatDate(date) {
        return date.toISOString().split('T')[0];
    }

    isSameDay(date1, date2) {
        return date1.getDate() === date2.getDate() &&
               date1.getMonth() === date2.getMonth() &&
               date1.getFullYear() === date2.getFullYear();
    }
}

// Initialize calendar when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new HealthCalendar();
});
