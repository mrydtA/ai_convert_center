/* ===================================
   TokenHub - 首页 JavaScript
   =================================== */

document.addEventListener('DOMContentLoaded', function() {
    // Model tabs filtering
    const tabButtons = document.querySelectorAll('.tab-btn');
    const modelCards = document.querySelectorAll('.model-card');
    
    tabButtons.forEach(button => {
        button.addEventListener('click', function() {
            const category = this.dataset.tab;
            
            // Update active button
            tabButtons.forEach(btn => btn.classList.remove('active'));
            this.classList.add('active');
            
            // Filter model cards with animation
            modelCards.forEach(card => {
                if (category === 'all' || card.dataset.category === category) {
                    card.style.display = 'block';
                    setTimeout(() => {
                        card.style.opacity = '1';
                        card.style.transform = 'translateY(0)';
                    }, 50);
                } else {
                    card.style.opacity = '0';
                    card.style.transform = 'translateY(20px)';
                    setTimeout(() => {
                        card.style.display = 'none';
                    }, 300);
                }
            });
        });
    });
    
    // Smooth scroll to sections
    document.querySelectorAll('.nav-link[href^="#"], .btn[href^="#"]').forEach(link => {
        link.addEventListener('click', function(e) {
            const href = this.getAttribute('href');
            if (href !== '#' && href.startsWith('#')) {
                e.preventDefault();
                const target = document.querySelector(href);
                if (target) {
                    const offsetTop = target.offsetTop - 70; // Account for fixed navbar
                    window.scrollTo({
                        top: offsetTop,
                        behavior: 'smooth'
                    });
                }
            }
        });
    });
    
    // Animate stats on scroll
    const statsSection = document.querySelector('.hero-stats');
    if (statsSection) {
        const statValues = statsSection.querySelectorAll('.stat-value');
        
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    statValues.forEach(stat => {
                        const finalValue = stat.textContent;
                        stat.textContent = '0';
                        animateValue(stat, finalValue);
                    });
                    observer.unobserve(entry.target);
                }
            });
        }, { threshold: 0.5 });
        
        observer.observe(statsSection);
    }
});

// Animate counter values
function animateValue(element, finalValue) {
    const isPercentage = finalValue.includes('%');
    const hasPlus = finalValue.includes('+');
    const hasCurrency = finalValue.includes('¥');
    const hasX = finalValue.includes('x');
    
    let numericValue = finalValue.replace(/[^0-9.]/g, '');
    const target = parseFloat(numericValue);
    const duration = 2000; // 2 seconds
    const steps = 60;
    const increment = target / steps;
    let current = 0;
    
    const timer = setInterval(() => {
        current += increment;
        if (current >= target) {
            current = target;
            clearInterval(timer);
        }
        
        let displayValue = current.toFixed(current < 10 ? 2 : 0);
        if (isPercentage) displayValue += '%';
        if (hasPlus && current === target) displayValue += '+';
        if (hasCurrency) displayValue = '¥' + displayValue;
        if (hasX && current === target) displayValue += 'x';
        
        element.textContent = displayValue;
    }, duration / steps);
}

// Add hover effect to feature cards
const featureCards = document.querySelectorAll('.feature-card');
featureCards.forEach(card => {
    card.addEventListener('mouseenter', function() {
        featureCards.forEach(c => {
            if (c !== this) {
                c.style.opacity = '0.7';
            }
        });
    });
    
    card.addEventListener('mouseleave', function() {
        featureCards.forEach(c => {
            c.style.opacity = '1';
        });
    });
});
