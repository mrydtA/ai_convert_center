/* ===================================
   TokenHub - 文档中心 JavaScript
   =================================== */

document.addEventListener('DOMContentLoaded', function() {
    // Active menu item based on scroll position
    const sections = document.querySelectorAll('.docs-section');
    const menuItems = document.querySelectorAll('.docs-menu-item');
    
    const observerOptions = {
        root: null,
        rootMargin: '-20% 0px -60% 0px',
        threshold: 0
    };
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                const id = entry.target.getAttribute('id');
                
                // Update active menu item
                menuItems.forEach(item => {
                    item.classList.remove('active');
                    if (item.getAttribute('href') === '#' + id) {
                        item.classList.add('active');
                    }
                });
            }
        });
    }, observerOptions);
    
    sections.forEach(section => observer.observe(section));
    
    // Smooth scroll for menu items
    menuItems.forEach(item => {
        item.addEventListener('click', function(e) {
            const href = this.getAttribute('href');
            if (href.startsWith('#')) {
                e.preventDefault();
                const target = document.querySelector(href);
                if (target) {
                    const offsetTop = target.offsetTop - 90; // Account for fixed navbar
                    window.scrollTo({
                        top: offsetTop,
                        behavior: 'smooth'
                    });
                    
                    // Update URL hash without scrolling
                    history.pushState(null, null, href);
                }
            }
        });
    });
    
    // Copy code functionality for docs
    const copyButtons = document.querySelectorAll('.copy-btn');
    copyButtons.forEach(button => {
        button.addEventListener('click', function() {
            const codeBlock = this.closest('.docs-code').querySelector('code');
            const text = codeBlock.textContent;
            
            navigator.clipboard.writeText(text).then(() => {
                const originalHTML = this.innerHTML;
                this.innerHTML = '<span>✅</span> 已复制';
                this.style.borderColor = 'var(--success-color)';
                this.style.color = 'var(--success-color)';
                
                setTimeout(() => {
                    this.innerHTML = originalHTML;
                    this.style.borderColor = '';
                    this.style.color = '';
                }, 2000);
            }).catch(err => {
                console.error('复制失败:', err);
                this.innerHTML = '<span>❌</span> 复制失败';
                
                setTimeout(() => {
                    this.innerHTML = '<span>📋</span> 复制';
                }, 2000);
            });
        });
    });
    
    // Search functionality (placeholder for future implementation)
    const searchInput = document.getElementById('docs-search');
    if (searchInput) {
        searchInput.addEventListener('input', function() {
            const query = this.value.toLowerCase();
            // Future: implement search filtering
            console.log('Search query:', query);
        });
    }
    
    // Highlight current section in TOC
    function highlightTOC() {
        const scrollPos = window.scrollY + 150;
        
        sections.forEach(section => {
            const sectionTop = section.offsetTop;
            const sectionHeight = section.offsetHeight;
            const sectionId = section.getAttribute('id');
            
            if (scrollPos >= sectionTop && scrollPos < sectionTop + sectionHeight) {
                menuItems.forEach(item => {
                    item.classList.remove('active');
                    if (item.getAttribute('href') === '#' + sectionId) {
                        item.classList.add('active');
                    }
                });
            }
        });
    }
    
    window.addEventListener('scroll', highlightTOC);
    highlightTOC(); // Initial call
});

// Table of contents click handler
function scrollToSection(id) {
    const element = document.getElementById(id);
    if (element) {
        const offsetTop = element.offsetTop - 90;
        window.scrollTo({
            top: offsetTop,
            behavior: 'smooth'
        });
    }
}
