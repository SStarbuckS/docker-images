// 防止主题闪烁：在页面渲染前立即应用主题
// 此脚本必须在 <head> 中同步加载（不使用 defer 或 async）
(function() {
    const theme = localStorage.getItem('theme') || 'auto';
    if (theme === 'auto') {
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        document.documentElement.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
    } else {
        document.documentElement.setAttribute('data-theme', theme);
    }
})(); 