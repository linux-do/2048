class ThemeToggle {
    constructor() {
        this.btn = document.getElementById('theme-toggle');
        if (!this.btn) return;
        this.applySaved();
        this.btn.addEventListener('click', () => this.toggle());
    }

    applySaved() {
        const saved = localStorage.getItem('theme');
        if (saved === 'dark') {
            document.body.classList.add('dark-mode');
            this.btn.textContent = 'Light Mode';
        } else {
            this.btn.textContent = 'Dark Mode';
        }
    }

    toggle() {
        if (document.body.classList.contains('dark-mode')) {
            document.body.classList.remove('dark-mode');
            localStorage.setItem('theme', 'light');
            this.btn.textContent = 'Dark Mode';
        } else {
            document.body.classList.add('dark-mode');
            localStorage.setItem('theme', 'dark');
            this.btn.textContent = 'Light Mode';
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    window.themeToggle = new ThemeToggle();
});
