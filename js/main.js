// PostHog initialisation (minified stub injected by PostHog)
!function(t,e){var o,n,p,r;e.__SV||(window.posthog&&window.posthog.__loaded)||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}p||((p=t.createElement("script")).type="text/javascript",p.crossOrigin="anonymous",p.async=!0,p.src=s.api_host.replace(".i.posthog.com","-assets.i.posthog.com")+"/static/array.js",p.onerror=function(){p=null},(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r));var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="fo po init Fo Oo qo Zs Lo Bo Ro capture Do vo Go calculateEventProperties Vo register register_once register_for_session unregister unregister_for_session Ko Ao Zo getFeatureFlag getFeatureFlagPayload getFeatureFlagResult getAllFeatureFlags isFeatureEnabled reloadFeatureFlags updateFlags updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures on onFeatureFlags onSurveysLoaded onSessionId getSurveys getActiveMatchingSurveys renderSurvey displaySurvey cancelPendingSurvey canRenderSurvey canRenderSurveyAsync Yo identify setPersonProperties unsetPersonProperties group resetGroups setPersonPropertiesForFlags resetPersonPropertiesForFlags setGroupPropertiesForFlags resetGroupPropertiesForFlags reset Xo shutdown setIdentity clearIdentity get_distinct_id getGroups get_session_id get_session_replay_url alias set_config startSessionRecording stopSessionRecording sessionRecordingStarted captureException addExceptionStep captureLog startExceptionAutocapture stopExceptionAutocapture loadToolbar get_property getSessionProperty Qo Uo createPersonProfile setInternalOrTestUser Jo Eo il opt_in_capturing opt_out_capturing has_opted_in_capturing has_opted_out_capturing get_explicit_consent_status is_capturing clear_opt_in_out_capturing Ho debug Js mn getPageViewId captureTraceFeedback captureTraceMetric Co".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);
posthog.init('phc_zugUWyUvq2VWid5q6XxqRtHnBXsoLDB9BhaaY6VxVuXJ', {
    api_host: 'https://metrics-api.jbirdkerr.net',
    ui_host: 'https://us.posthog.com',
    person_profiles: 'always',
    cross_subdomain_cookie: false,
    request_batching: true,
    batch_size: 15,          // Flushes immediately after 15 quick clicks
    flush_interval_ms: 2000,  // Or flushes every 1 second max
    disable_session_recording: true
});

function captureEvent(eventName, properties = {}) {
    if (window.posthog) {
        window.posthog.capture(eventName, properties);
    }
}

(function() {
    /* ============================================================
       MODAL SYSTEM (HTMX-driven)
       ============================================================ */
    const metricsToggle = document.getElementById('metrics-toggle');
    const creditsToggle = document.getElementById('credits-toggle');
    const metricsModal = document.getElementById('metrics-modal');
    const creditsModal = document.getElementById('credits-modal');
    let metricsPollInterval = null;

    function openMetricsModal() {
        if (typeof flushEyeDistance === 'function') {
            flushEyeDistance();
        }
        metricsModal.classList.add('active');
        if (metricsPollInterval) {
            clearInterval(metricsPollInterval);
        }
        window.htmx.ajax('GET', 'https://metrics-api.jbirdkerr.net/metrics', '#metrics-container');
        metricsPollInterval = setInterval(() => {
            if (document.hidden || !metricsModal.classList.contains('active')) {
                return;
            }
            window.htmx.ajax('GET', 'https://metrics-api.jbirdkerr.net/metrics', '#metrics-container');
        }, 5000);
    }

    function closeMetricsModal() {
        metricsModal.classList.remove('active');
        if (metricsPollInterval) {
            clearInterval(metricsPollInterval);
            metricsPollInterval = null;
        }
    }

    function openCreditsModal() {
        creditsModal.classList.add('active');
    }

    function closeCreditsModal() {
        creditsModal.classList.remove('active');
    }

    // Modal triggers
    metricsToggle.addEventListener('click', (e) => {
        e.preventDefault();
        openMetricsModal();
    });

    const metricsCloseBtn = metricsModal.querySelector('.modal-close');
    if (metricsCloseBtn) {
        metricsCloseBtn.addEventListener('click', closeMetricsModal);
    }

    creditsToggle.addEventListener('click', openCreditsModal);
    const creditsCloseBtn = creditsModal.querySelector('.modal-close');
    if (creditsCloseBtn) {
        creditsCloseBtn.addEventListener('click', closeCreditsModal);
    }

    // Close modals on overlay click
    document.addEventListener('click', (e) => {
        if (e.target === metricsModal) {
            closeMetricsModal();
        } else if (e.target === creditsModal) {
            closeCreditsModal();
        }
    });

    // Format metrics timestamps into local browser timezone
    document.addEventListener('htmx:afterSwap', (e) => {
        if (e.detail && e.detail.target && e.detail.target.id === 'metrics-container') {
            const timeEl = e.detail.target.querySelector('.metrics-timestamp');
            if (timeEl && timeEl.dataset.utc && timeEl.dataset.utc !== '—') {
                const date = new Date(timeEl.dataset.utc);
                if (!isNaN(date.getTime())) {
                    timeEl.textContent = date.toLocaleTimeString([], {
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit',
                        hour12: false
                    });
                }
            }
        }
    });

    // Pause polling when tab is hidden
    document.addEventListener('visibilitychange', () => {
        if (document.hidden && metricsPollInterval) {
            clearInterval(metricsPollInterval);
            metricsPollInterval = null;
        } else if (!document.hidden && metricsModal.classList.contains('active')) {
            openMetricsModal();
        }
    });
    /* ============================================================
       AUDIO ENGINE (Web Audio API Synthesizer - Zero Dependencies)
       ============================================================ */
    let audioCtx = null;
    function getAudioContext() {
        if (!audioCtx) {
            const AudioContext = window.AudioContext || window.webkitAudioContext;
            if (AudioContext) {
                audioCtx = new AudioContext();
            }
        }
        if (audioCtx && audioCtx.state === 'suspended') {
            audioCtx.resume();
        }
        return audioCtx;
    }
    // Squeak / Boop Sound
    function playBoopSound(pitchMod = 1) {
        const ctx = getAudioContext();
        if (!ctx) return;
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        const now = ctx.currentTime;
        const baseFreq = 540 * pitchMod;
        osc.type = 'sine';
        osc.frequency.setValueAtTime(baseFreq, now);
        osc.frequency.exponentialRampToValueAtTime(baseFreq * 1.6, now + 0.06);
        osc.frequency.exponentialRampToValueAtTime(baseFreq * 0.8, now + 0.15);
        gain.gain.setValueAtTime(0.3, now);
        gain.gain.exponentialRampToValueAtTime(0.001, now + 0.16);
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.start(now);
        osc.stop(now + 0.17);
    }
    // Dog Woof / Bark Sound
    function playBarkSound() {
        const ctx = getAudioContext();
        if (!ctx) return;
        const now = ctx.currentTime;
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        const filter = ctx.createBiquadFilter();
        osc.type = 'sawtooth';
        osc.frequency.setValueAtTime(320, now);
        osc.frequency.exponentialRampToValueAtTime(110, now + 0.22);
        filter.type = 'bandpass';
        filter.frequency.setValueAtTime(450, now);
        filter.Q.setValueAtTime(3, now);
        gain.gain.setValueAtTime(0.001, now);
        gain.gain.linearRampToValueAtTime(0.45, now + 0.03);
        gain.gain.exponentialRampToValueAtTime(0.001, now + 0.25);
        osc.connect(filter);
        filter.connect(gain);
        gain.connect(ctx.destination);
        osc.start(now);
        osc.stop(now + 0.26);
    }
    // 8-Bit Party Fanfare Chime
    function playPartyFanfare() {
        const ctx = getAudioContext();
        if (!ctx) return;
        const notes = [
            { f: 523.25, d: 0.08 }, // C5
            { f: 659.25, d: 0.08 }, // E5
            { f: 783.99, d: 0.08 }, // G5
            { f: 1046.50, d: 0.24 } // C6
        ];
        let time = ctx.currentTime;
        notes.forEach(n => {
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            osc.type = 'square';
            osc.frequency.setValueAtTime(n.f, time);
            gain.gain.setValueAtTime(0.12, time);
            gain.gain.exponentialRampToValueAtTime(0.001, time + n.d);
            osc.connect(gain);
            gain.connect(ctx.destination);
            osc.start(time);
            osc.stop(time + n.d + 0.02);
            time += n.d * 0.9;
        });
    }
    // Googly Wobble Sound
    function playGooglySound() {
        const ctx = getAudioContext();
        if (!ctx) return;
        const now = ctx.currentTime;
        const osc = ctx.createOscillator();
        const gain = ctx.createGain();
        osc.type = 'triangle';
        osc.frequency.setValueAtTime(240, now);
        osc.frequency.linearRampToValueAtTime(600, now + 0.09);
        osc.frequency.linearRampToValueAtTime(320, now + 0.18);
        osc.frequency.linearRampToValueAtTime(700, now + 0.26);
        gain.gain.setValueAtTime(0.2, now);
        gain.gain.exponentialRampToValueAtTime(0.001, now + 0.3);
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.start(now);
        osc.stop(now + 0.32);
    }
    /* ============================================================
       BOOP & PARTICLE SYSTEM
       ============================================================ */
    let boopCount = 0;
    const photoEl = document.getElementById('photo');
    const snootEl = document.getElementById('snoot');
    const stageEl = document.getElementById('stage');
    const boopPhrases = [
        "BOOP! 🐶",
        "SNOOT BOOPED! ✨",
        "10/10 GOOD BOY 🦴",
        "SUCH BEARD! ❤️",
        "WOOF! 🐾",
        "HENRY APPROVED 👍",
        "MEGA BOOP! ⚡",
        "BOOP × "
    ];
    const particleIcons = ["🐾", "❤️", "🦴", "✨", "🐶", "🍖", "💖", "⭐"];
    function spawnParticles(x, y, count = 7) {
        for (let i = 0; i < count; i++) {
            const particle = document.createElement('div');
            particle.className = 'floating-particle';
            particle.textContent = particleIcons[Math.floor(Math.random() * particleIcons.length)];
            const angle = (Math.PI * 2 * i) / count + (Math.random() - 0.5) * 0.5;
            const distance = 60 + Math.random() * 80;
            const dx = Math.cos(angle) * distance;
            const dy = -Math.abs(Math.sin(angle) * distance) - 40;
            const rot = (Math.random() - 0.5) * 90 + 'deg';
            particle.style.left = `${x}px`;
            particle.style.top = `${y}px`;
            particle.style.setProperty('--dx', `${dx}px`);
            particle.style.setProperty('--dy', `${dy}px`);
            particle.style.setProperty('--rot', rot);
            document.body.appendChild(particle);
            setTimeout(() => particle.remove(), 1200);
        }
    }
    function spawnBoopBadge(x, y, text) {
        const badge = document.createElement('div');
        badge.className = 'boop-badge';
        badge.textContent = text;
        badge.style.left = `${x}px`;
        badge.style.top = `${y}px`;
        document.body.appendChild(badge);
        setTimeout(() => badge.remove(), 1000);
    }
    function triggerBoop(e) {
        boopCount++;
        const pitch = 0.85 + Math.random() * 0.4;
        playBoopSound(pitch);
        // Track with PostHog
        captureEvent('boop', {
            combo_count: boopCount,
            via: e ? 'click' : 'api'
        });
        photoEl.classList.remove('boop-squish');
        void photoEl.offsetWidth;
        photoEl.classList.add('boop-squish');
        setTimeout(() => photoEl.classList.remove('boop-squish'), 140);
        let x, y;
        if (e && e.clientX && e.clientY) {
            x = e.clientX;
            y = e.clientY;
        } else {
            const rect = snootEl.getBoundingClientRect();
            x = rect.left + rect.width / 2;
            y = rect.top + rect.height / 2;
        }
        spawnParticles(x, y, 8);
        let phrase;
        if (boopCount % 10 === 0) {
            phrase = `🔥 COMBO × ${boopCount}! 🔥`;
        } else if (boopCount > 5 && Math.random() > 0.6) {
            phrase = `BOOP × ${boopCount}`;
        } else {
            phrase = boopPhrases[Math.floor(Math.random() * (boopPhrases.length - 1))];
        }
        spawnBoopBadge(x, y - 20, phrase);
    }
    snootEl.addEventListener('click', (e) => {
        e.stopPropagation();
        triggerBoop(e);
    });
    /* ============================================================
       EYE TRACKING CONTROLLER
       ============================================================ */
    const socketLeft = document.getElementById('socket-left');
    const socketRight = document.getElementById('socket-right');
    const irisLeft = document.getElementById('iris-left');
    const irisRight = document.getElementById('iris-right');
    let eyeTrackingActive = false;
    let lastPointerX = window.innerWidth / 2;
    let lastPointerY = window.innerHeight / 2;
    // Eye movement distance tracking
    let lastEyeLeftX = 0, lastEyeLeftY = 0;
    let accumulatedEyeDistance = 0;
    let lastEyeFlushTime = Date.now();
    function recordEyeMovement(newX, newY) {
        if (lastEyeLeftX !== 0 || lastEyeLeftY !== 0) {
            const delta = Math.hypot(newX - lastEyeLeftX, newY - lastEyeLeftY);
            if (delta > 0.5 && delta < 50) { // filter out jumps/toggles
                accumulatedEyeDistance += delta;
            }
        }
        lastEyeLeftX = newX;
        lastEyeLeftY = newY;
        // Flush every 3 seconds if moved more than 10 pixels
        const now = Date.now();
        if (now - lastEyeFlushTime > 3000 && accumulatedEyeDistance >= 10) {
            flushEyeDistance();
        }
    }
    function flushEyeDistance() {
        if (accumulatedEyeDistance >= 5) {
            captureEvent('eye_movement', {
                distance_px: Math.round(accumulatedEyeDistance)
            });
            accumulatedEyeDistance = 0;
            lastEyeFlushTime = Date.now();
        }
    }
    window.addEventListener('beforeunload', flushEyeDistance);
    window.addEventListener('pagehide', flushEyeDistance);
    let googlyActive = false;
    const eyeLeft = document.getElementById('eye-left');
    const eyeRight = document.getElementById('eye-right');
    const pupilLeft = document.getElementById('pupil-left');
    const pupilRight = document.getElementById('pupil-right');
    function toggleEyeTracking(forceState) {
        eyeTrackingActive = (forceState !== undefined) ? forceState : !eyeTrackingActive;
        if (eyeTrackingActive) {
            stageEl.classList.add('tracking-active');
            updateRealisticEye(irisLeft, socketLeft, lastPointerX, lastPointerY);
            updateRealisticEye(irisRight, socketRight, lastPointerX, lastPointerY);
            captureEvent('feature_toggle', { feature: 'eye_tracking', enabled: true, trigger: 'click' });
            console.log("%c👀 Eye tracking activated!", "color: #ffeb3b; font-weight: bold;");
        } else {
            stageEl.classList.remove('tracking-active');
            if (irisLeft) irisLeft.style.transform = 'translate(0px, 0px)';
            if (irisRight) irisRight.style.transform = 'translate(0px, 0px)';
            flushEyeDistance();
            lastEyeLeftX = 0;
            lastEyeLeftY = 0;
            captureEvent('feature_toggle', { feature: 'eye_tracking', enabled: false });
            console.log("%c👀 Eyes returned to rest.", "color: #888;");
        }
        return eyeTrackingActive;
    }
    photoEl.addEventListener('click', (e) => {
        toggleEyeTracking();
    });
    function updateRealisticEye(irisEl, socketEl, targetX, targetY) {
        if (!irisEl || !socketEl) return;
        const rect = socketEl.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) return;
        const centerX = rect.left + rect.width / 2;
        const centerY = rect.top + rect.height / 2;
        const dx = targetX - centerX;
        const dy = targetY - centerY;
        const distance = Math.hypot(dx, dy);
        const angle = Math.atan2(dy, dx);
        const maxMoveX = rect.width * 0.28;
        const maxMoveY = rect.height * 0.32;
        const maxScreenDist = Math.max(window.innerWidth, window.innerHeight) * 0.45;
        const intensity = Math.min(distance / maxScreenDist, 1.0);
        const mx = Math.cos(angle) * maxMoveX * intensity;
        const my = Math.sin(angle) * maxMoveY * intensity;
        irisEl.style.transform = `translate(${mx}px, ${my}px)`;
        if (irisEl === irisLeft) {
            recordEyeMovement(mx, my);
        }
    }
    function updatePupilPosition(pupilEl, eyeEl, targetX, targetY) {
        if (!pupilEl || !eyeEl) return;
        const rect = eyeEl.getBoundingClientRect();
        const eyeCenterX = rect.left + rect.width / 2;
        const eyeCenterY = rect.top + rect.height / 2;
        const dx = targetX - eyeCenterX;
        const dy = targetY - eyeCenterY;
        const distance = Math.hypot(dx, dy);
        const maxOffset = rect.width * 0.22;
        const angle = Math.atan2(dy, dx);
        const offsetDist = Math.min(distance * 0.08, maxOffset);
        const pupilX = Math.cos(angle) * offsetDist;
        const pupilY = Math.sin(angle) * offsetDist;
        pupilEl.style.transform = `translate(${pupilX}px, ${pupilY}px)`;
    }
    function onPointerMove(clientX, clientY) {
        lastPointerX = clientX;
        lastPointerY = clientY;
        if (eyeTrackingActive) {
            updateRealisticEye(irisLeft, socketLeft, clientX, clientY);
            updateRealisticEye(irisRight, socketRight, clientX, clientY);
        }
        if (googlyActive) {
            updatePupilPosition(pupilLeft, eyeLeft, clientX, clientY);
            updatePupilPosition(pupilRight, eyeRight, clientX, clientY);
        }
    }
    window.addEventListener('pointermove', (e) => {
        onPointerMove(e.clientX, e.clientY);
    });
    window.addEventListener('touchmove', (e) => {
        if (e.touches && e.touches[0]) {
            onPointerMove(e.touches[0].clientX, e.touches[0].clientY);
        }
    }, { passive: true });
    window.addEventListener('touchstart', (e) => {
        if (e.touches && e.touches[0]) {
            onPointerMove(e.touches[0].clientX, e.touches[0].clientY);
        }
    }, { passive: true });
    function toggleGooglyEyes(forceState) {
        googlyActive = (forceState !== undefined) ? forceState : !googlyActive;
        if (googlyActive) {
            playGooglySound();
            eyeLeft.classList.add('active');
            eyeRight.classList.add('active');
            captureEvent('feature_toggle', { feature: 'googly_eyes', enabled: true, trigger: 'double_click' });
            console.log("%c👀 Googly eyes ACTIVATED!", "color: #00e5ff; font-weight: bold;");
        } else {
            eyeLeft.classList.remove('active');
            eyeRight.classList.remove('active');
            captureEvent('feature_toggle', { feature: 'googly_eyes', enabled: false });
            console.log("%c👀 Googly eyes removed.", "color: #888;");
        }
        return googlyActive;
    }
    let lastTapTime = 0;
    window.addEventListener('dblclick', (e) => {
        toggleGooglyEyes();
    });
    window.addEventListener('touchend', (e) => {
        const now = Date.now();
        if (now - lastTapTime < 320 && now - lastTapTime > 0) {
            toggleGooglyEyes();
            lastTapTime = 0;
        } else {
            lastTapTime = now;
        }
    });
    /* ============================================================
       PARTY MODE & CONFETTI
       ============================================================ */
    let partyActive = false;
    const sunglassesEl = document.getElementById('sunglasses');
    const partyBannerEl = document.getElementById('party-banner');
    const fxCanvas = document.getElementById('fx-canvas');
    const ctx2d = fxCanvas.getContext('2d');
    let confettiParticles = [];
    let animFrameId = null;
    function resizeCanvas() {
        fxCanvas.width = window.innerWidth;
        fxCanvas.height = window.innerHeight;
    }
    window.addEventListener('resize', resizeCanvas);
    resizeCanvas();
    function spawnConfetti(count = 120) {
        const colors = ['#ff0055', '#00f0ff', '#ffeb3b', '#00ff88', '#ff00ff', '#ffffff'];
        for (let i = 0; i < count; i++) {
            confettiParticles.push({
                x: Math.random() * fxCanvas.width,
                y: -20 - Math.random() * 100,
                size: 6 + Math.random() * 8,
                color: colors[Math.floor(Math.random() * colors.length)],
                vx: (Math.random() - 0.5) * 4,
                vy: 2 + Math.random() * 4,
                rotation: Math.random() * Math.PI * 2,
                vRot: (Math.random() - 0.5) * 0.2
            });
        }
        if (!animFrameId) {
            renderConfetti();
        }
    }
    function renderConfetti() {
        ctx2d.clearRect(0, 0, fxCanvas.width, fxCanvas.height);
        for (let i = confettiParticles.length - 1; i >= 0; i--) {
            const p = confettiParticles[i];
            p.x += p.vx;
            p.y += p.vy;
            p.rotation += p.vRot;
            ctx2d.save();
            ctx2d.translate(p.x, p.y);
            ctx2d.rotate(p.rotation);
            ctx2d.fillStyle = p.color;
            ctx2d.fillRect(-p.size / 2, -p.size / 2, p.size, p.size * 1.5);
            ctx2d.restore();
            if (p.y > fxCanvas.height + 20) {
                confettiParticles.splice(i, 1);
            }
        }
        if (confettiParticles.length > 0 || partyActive) {
            animFrameId = requestAnimationFrame(renderConfetti);
        } else {
            animFrameId = null;
            ctx2d.clearRect(0, 0, fxCanvas.width, fxCanvas.height);
        }
    }
    function togglePartyMode(forceState) {
        partyActive = (forceState !== undefined) ? forceState : !partyActive;
        if (partyActive) {
            document.body.classList.add('party-mode');
            sunglassesEl.classList.add('active');
            partyBannerEl.classList.add('active');
            playPartyFanfare();
            spawnConfetti(150);
            captureEvent('feature_toggle', { feature: 'party_mode', enabled: true, trigger: 'key_p' });
            console.log("%c🎉 PARTY MODE ENGAGED!", "color: #ff007f; font-weight: bold; font-size: 1.2rem;");
        } else {
            document.body.classList.remove('party-mode');
            sunglassesEl.classList.remove('active');
            partyBannerEl.classList.remove('active');
            captureEvent('feature_toggle', { feature: 'party_mode', enabled: false });
            console.log("%c🎉 Party mode disabled.", "color: #888;");
        }
        return partyActive;
    }
    /* ============================================================
       KEYBOARD SHORTCUTS & SECRET COMMANDS
       ============================================================ */
    window.addEventListener('keydown', (e) => {
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') return;
        const key = e.key.toLowerCase();
        if (key === 'b') {
            triggerBoop();
        } else if (key === 'e') {
            toggleEyeTracking();
        } else if (key === 'g') {
            toggleGooglyEyes();
        } else if (key === 'p') {
            togglePartyMode();
        } else if (key === 'w') {
            playBarkSound();
            captureEvent('bark', { trigger: 'key_w' });
        } else if (key === 'm') {
            if (metricsModal.classList.contains('active')) {
                closeMetricsModal();
            } else {
                openMetricsModal();
            }
        } else if (key === 'escape') {
            closeMetricsModal();
            closeCreditsModal();
        }
    });
    /* Expose Global Helper API for Interactive Exploration */
    window.HenryBeard = {
        boop: () => triggerBoop(),
        bark: () => playBarkSound(),
        toggleEyes: (s) => toggleEyeTracking(s),
        toggleGoogly: (s) => toggleGooglyEyes(s),
        party: (s) => togglePartyMode(s)
    };
    console.log("%c🐶 Woof! Henry Beard Interactive Canvas Ready.", "color: #00d4ff; font-weight: bold; font-size: 1.1rem;");
    console.log("%cHotkeys: [B] Boop • [E] Toggle Eyes • [G] Googly Eyes • [P] Party Mode • [W] Woof • [M] Metrics", "color: #aaa;");
})();
