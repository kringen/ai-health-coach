// Dashboard functionality
class HealthDashboard {
    constructor() {
        this.charts = {};
        this.currentRange = 30; // Default to 30 days
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadData();
    }

    bindEvents() {
        const timeRange = document.getElementById('timeRange');
        if (timeRange) {
            timeRange.addEventListener('change', (e) => {
                this.currentRange = parseInt(e.target.value);
                this.loadData();
            });
        }
    }

    async loadData() {
        this.showLoading(true);
        
        try {
            // Load summary statistics
            await this.loadSummary();
            
            // Load trend data
            await this.loadTrends();
            
        } catch (error) {
            console.error('Error loading dashboard data:', error);
        } finally {
            this.showLoading(false);
        }
    }

    async loadSummary() {
        try {
            const response = await fetch(`/api/data/summary?days=${this.currentRange}`, {
                headers: this.getAuthHeaders()
            });

            if (response.ok) {
                const summary = await response.json();
                this.updateSummaryCards(summary);
            }
        } catch (error) {
            console.error('Error loading summary:', error);
        }
    }

    async loadTrends() {
        try {
            const response = await fetch(`/api/data/trends?days=${this.currentRange}`, {
                headers: this.getAuthHeaders()
            });

            if (response.ok) {
                const trends = await response.json();
                this.updateCharts(trends);
            }
        } catch (error) {
            console.error('Error loading trends:', error);
        }
    }

    updateSummaryCards(summary) {
        // Update average water
        const avgWater = document.getElementById('avgWater');
        if (avgWater) {
            avgWater.textContent = `${summary.average_water.toFixed(1)}L`;
        }

        // Update average sleep
        const avgSleep = document.getElementById('avgSleep');
        if (avgSleep) {
            avgSleep.textContent = `${summary.average_sleep.toFixed(1)}/5`;
        }

        // Update exercise days (count from common exercises)
        const exerciseDays = document.getElementById('exerciseDays');
        if (exerciseDays) {
            const exerciseCount = summary.common_exercises ? summary.common_exercises.length : 0;
            exerciseDays.textContent = exerciseCount.toString();
        }

        // Update total entries
        const totalEntries = document.getElementById('totalEntries');
        if (totalEntries) {
            totalEntries.textContent = summary.total_entries.toString();
        }
    }

    updateCharts(trends) {
        this.createWaterChart(trends);
        this.createSleepChart(trends);
        this.createExerciseChart(trends);
        this.createSummaryChart(trends);
    }

    createWaterChart(trends) {
        const ctx = document.getElementById('waterChart');
        if (!ctx) return;

        if (this.charts.water) {
            this.charts.water.destroy();
        }

        const labels = trends.map(t => new Date(t.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }));
        const data = trends.map(t => t.water_intake);

        this.charts.water = new Chart(ctx, {
            type: 'line',
            data: {
                labels: labels,
                datasets: [{
                    label: 'Water Intake (L)',
                    data: data,
                    borderColor: '#3b82f6',
                    backgroundColor: 'rgba(59, 130, 246, 0.1)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        title: {
                            display: true,
                            text: 'Liters'
                        }
                    }
                }
            }
        });
    }

    createSleepChart(trends) {
        const ctx = document.getElementById('sleepChart');
        if (!ctx) return;

        if (this.charts.sleep) {
            this.charts.sleep.destroy();
        }

        const labels = trends.map(t => new Date(t.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }));
        const data = trends.map(t => t.sleep_quality || 0);

        this.charts.sleep = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: labels,
                datasets: [{
                    label: 'Sleep Quality',
                    data: data,
                    backgroundColor: '#8b5cf6',
                    borderRadius: 4
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 5,
                        title: {
                            display: true,
                            text: 'Quality (1-5)'
                        }
                    }
                }
            }
        });
    }

    createExerciseChart(trends) {
        const ctx = document.getElementById('exerciseChart');
        if (!ctx) return;

        if (this.charts.exercise) {
            this.charts.exercise.destroy();
        }

        const labels = trends.map(t => new Date(t.date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }));
        const data = trends.map(t => t.has_exercise ? 1 : 0);

        this.charts.exercise = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: labels,
                datasets: [{
                    label: 'Exercise',
                    data: data,
                    backgroundColor: data.map(d => d ? '#f59e0b' : '#e5e7eb'),
                    borderRadius: 4
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        display: false
                    }
                },
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 1,
                        ticks: {
                            stepSize: 1,
                            callback: function(value) {
                                return value === 1 ? 'Yes' : 'No';
                            }
                        },
                        title: {
                            display: true,
                            text: 'Exercise Done'
                        }
                    }
                }
            }
        });
    }

    createSummaryChart(trends) {
        const ctx = document.getElementById('summaryChart');
        if (!ctx) return;

        if (this.charts.summary) {
            this.charts.summary.destroy();
        }

        // Group data by week
        const weeklyData = this.groupByWeek(trends);
        const labels = weeklyData.map(w => w.week);
        
        this.charts.summary = new Chart(ctx, {
            type: 'radar',
            data: {
                labels: ['Water', 'Sleep', 'Exercise', 'Consistency'],
                datasets: weeklyData.map((week, index) => ({
                    label: week.week,
                    data: [
                        week.avgWater * 20, // Scale to 0-100
                        week.avgSleep * 20, // Scale to 0-100
                        week.exercisePercent,
                        week.consistency
                    ],
                    borderColor: this.getColor(index),
                    backgroundColor: this.getColor(index, 0.1),
                    pointBackgroundColor: this.getColor(index)
                }))
            },
            options: {
                responsive: true,
                scales: {
                    r: {
                        beginAtZero: true,
                        max: 100
                    }
                }
            }
        });
    }

    groupByWeek(trends) {
        const weeks = [];
        const weekSize = 7;
        
        for (let i = 0; i < trends.length; i += weekSize) {
            const weekData = trends.slice(i, i + weekSize);
            const startDate = new Date(weekData[0].date);
            const endDate = new Date(weekData[weekData.length - 1].date);
            
            const avgWater = weekData.reduce((sum, d) => sum + d.water_intake, 0) / weekData.length;
            const avgSleep = weekData.reduce((sum, d) => sum + (d.sleep_quality || 0), 0) / weekData.length;
            const exerciseCount = weekData.filter(d => d.has_exercise).length;
            const exercisePercent = (exerciseCount / weekData.length) * 100;
            const consistency = weekData.length === weekSize ? 100 : (weekData.length / weekSize) * 100;
            
            weeks.push({
                week: `${startDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} - ${endDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`,
                avgWater,
                avgSleep,
                exercisePercent,
                consistency
            });
        }
        
        return weeks.slice(-4); // Last 4 weeks
    }

    getColor(index, alpha = 1) {
        const colors = [
            `rgba(59, 130, 246, ${alpha})`,   // Blue
            `rgba(16, 185, 129, ${alpha})`,   // Green
            `rgba(245, 158, 11, ${alpha})`,   // Orange
            `rgba(139, 92, 246, ${alpha})`    // Purple
        ];
        return colors[index % colors.length];
    }

    showLoading(show) {
        const loading = document.getElementById('loading');
        if (loading) {
            loading.style.display = show ? 'flex' : 'none';
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
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new HealthDashboard();
});
