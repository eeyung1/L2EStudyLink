(() => {document.getElementById('readingTheme')?.addEventListener('click',()=>{const dark=document.documentElement.classList.toggle('dark');localStorage.setItem('darkMode',String(dark));});})();
