/* ===========================
   首页 JS 逻辑
   =========================== */

/* ===== 1. Header 滚动效果 ===== */
const header = document.querySelector(".header");

window.addEventListener("scroll", () => {
    if (window.scrollY > 50) {
        header.style.boxShadow = "0 6px 30px rgba(0,0,0,.08)";
        header.style.background = "rgba(255,255,255,0.98)";
    } else {
        header.style.boxShadow = "none";
        header.style.background = "rgba(255,255,255,0.95)";
    }
});

/* ===== 2. 平滑滚动 ===== */
document.querySelectorAll('a[href^="#"]').forEach(link => {
    link.addEventListener("click", e => {
        const targetId = link.getAttribute("href");
        if (targetId.length > 1) {
            const target = document.querySelector(targetId);
            if (target) {
                e.preventDefault();
                window.scrollTo({
                    top: target.offsetTop - 60,
                    behavior: "smooth"
                });
            }
        }
    });
});

/* ===== 3. 导航高亮 ===== */
const sections = document.querySelectorAll("section");
const navLinks = document.querySelectorAll(".nav a");

window.addEventListener("scroll", () => {
    let current = "";

    sections.forEach(section => {
        const sectionTop = section.offsetTop - 80;
        if (scrollY >= sectionTop) {
            current = section.getAttribute("id");
        }
    });

    navLinks.forEach(link => {
        link.classList.remove("active");
        if (link.getAttribute("href") === `#${current}`) {
            link.classList.add("active");
        }
    });
});

/* ===== 4. 页面进入动画 ===== */
const observer = new IntersectionObserver(entries => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.classList.add("show");
        }
    });
}, { threshold: 0.2 });

document.querySelectorAll(".section, .job-card, .culture-card").forEach(el => {
    el.classList.add("hidden");
    observer.observe(el);
});

/* ===== 5. 简历投递前拦截（示例） ===== */
document.querySelectorAll(".job-card a").forEach(btn => {
    btn.addEventListener("click", e => {
        const isLogin = false; // 后端登录态判断（占位）
        if (!isLogin) {
            e.preventDefault();
            alert("请先登录后再投递简历");
            window.location.href = "/page/login";
        }
    });
});
