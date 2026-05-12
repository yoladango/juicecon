(function() {
    'use strict';

    // Elements
    var loadingEl = document.getElementById('loading');
    var errorEl = document.getElementById('error');
    var errorMessageEl = document.getElementById('error-message');
    var displayEl = document.getElementById('display');
    var meterChipEl = document.getElementById('meter-chip');
    var meterReadoutValueEl = document.getElementById('meter-readout-value');
    var meterNeedleEl = document.getElementById('meter-needle');
    var meterBandsEl = document.getElementById('meter-bands');
    var descriptorEl = document.getElementById('descriptor');
    var descriptionEl = document.getElementById('description');
    var protocolEl = document.getElementById('protocol');
    var temperatureEl = document.getElementById('temperature');
    var dewpointEl = document.getElementById('dewpoint');
    var locationEl = document.getElementById('location');
    var stationEl = document.getElementById('station');
    var timestampEl = document.getElementById('timestamp');
    var refreshBtn = document.getElementById('refresh-btn');
    var locationBtn = document.getElementById('location-btn');
    var retryBtn = document.getElementById('retry-btn');
    var modalEl = document.getElementById('location-modal');
    var zipInput = document.getElementById('zip-input');
    var modalCancel = document.getElementById('modal-cancel');
    var modalSubmit = document.getElementById('modal-submit');
    var infoBtn = document.getElementById('info-btn');
    var infoModal = document.getElementById('info-modal');
    var infoClose = document.getElementById('info-close');
    var shareBtn = document.getElementById('share-btn');
    var refreshStatusEl = document.getElementById('refresh-status');
    var refreshStatusTextEl = document.getElementById('refresh-status-text');
    var timecodeEl = document.getElementById('timecode');
    var headerDateEl = document.getElementById('header-date');

    // State
    var currentLat = null;
    var currentLon = null;
    var currentZip = null;
    var autoRefreshTimer = null;
    var lastRefreshTime = null;
    var modalTriggerElement = null;
    var currentData = null;

    var AUTO_REFRESH_INTERVAL = 15 * 60 * 1000; // 15 minutes

    // System display names
    var systemNames = {
        juicecon: 'JUICECON',
        ccf: 'CUNNINGHAM CRACKLE FACTOR',
        none: 'DEWCON'
    };

    // Meter geometry — must match the SVG viewBox in index.html and CSS transform-origin in style.css
    var METER = {
        cx: 210,
        cy: 200,
        r: 160
    };

    // 11 equal-width segments. Ordered worst-dryness → comfort → worst-humidity (left to right).
    // Each id corresponds to a CSS class (.band-<id>) and a [data-system][data-level] state.
    var METER_SEGMENTS = [
        { id: 'ccf-1',   system: 'ccf',      level: 1, numeral: '1' },
        { id: 'ccf-2',   system: 'ccf',      level: 2, numeral: '2' },
        { id: 'ccf-3',   system: 'ccf',      level: 3, numeral: '3' },
        { id: 'ccf-4',   system: 'ccf',      level: 4, numeral: '4' },
        { id: 'ccf-5',   system: 'ccf',      level: 5, numeral: '5' },
        { id: 'comfort', system: 'none',     level: 'clear', numeral: '◆' },
        { id: 'jc-5',    system: 'juicecon', level: 5, numeral: '5' },
        { id: 'jc-4',    system: 'juicecon', level: 4, numeral: '4' },
        { id: 'jc-3',    system: 'juicecon', level: 3, numeral: '3' },
        { id: 'jc-2',    system: 'juicecon', level: 2, numeral: '2' },
        { id: 'jc-1',    system: 'juicecon', level: 1, numeral: '1' }
    ];

    var SVG_NS = 'http://www.w3.org/2000/svg';

    // Check for test mode dewpoint override via URL param
    var urlParams = new URLSearchParams(window.location.search);
    var testDewpoint = urlParams.get('_dewpoint');

    // Initialize
    function init() {
        decoratePanels();
        bindEvents();
        buildMeter();
        startTimecode();
        if (testDewpoint !== null) {
            fetchTestDewpoint(testDewpoint);
        } else {
            getLocation();
        }
    }

    // Add the small copper inspection square in the bottom-left of
    // each panel. Pure decoration; safe to fail if .panel isn't in
    // the DOM yet.
    function decoratePanels() {
        var panels = document.querySelectorAll('.panel');
        for (var i = 0; i < panels.length; i++) {
            if (panels[i].querySelector('.inspection-mark')) continue;
            var mark = document.createElement('span');
            mark.className = 'inspection-mark';
            mark.setAttribute('aria-hidden', 'true');
            panels[i].appendChild(mark);
        }
    }

    // Live UTC timecode in classification bar + header date
    function startTimecode() {
        function pad(n) { return n < 10 ? '0' + n : '' + n; }
        function tick() {
            var d = new Date();
            if (timecodeEl) {
                timecodeEl.textContent =
                    pad(d.getUTCHours()) + ':' +
                    pad(d.getUTCMinutes()) + ':' +
                    pad(d.getUTCSeconds()) + ' UTC';
            }
            if (headerDateEl) {
                headerDateEl.textContent =
                    d.getUTCFullYear() + '.' +
                    pad(d.getUTCMonth() + 1) + '.' +
                    pad(d.getUTCDate()) + 'Z';
            }
        }
        tick();
        setInterval(tick, 1000);
    }

    // =====================
    // Meter (VU gauge)
    // =====================

    // Convert fractional position along the half-arc [0..1] to a point on a given radius
    function pointOnArcT(t, radius) {
        if (radius == null) radius = METER.r;
        var angleRad = Math.PI * (1 - t);
        return {
            x: METER.cx + radius * Math.cos(angleRad),
            y: METER.cy - radius * Math.sin(angleRad)
        };
    }

    function arcPathT(t1, t2) {
        var p1 = pointOnArcT(t1);
        var p2 = pointOnArcT(t2);
        return 'M ' + p1.x.toFixed(2) + ' ' + p1.y.toFixed(2) +
               ' A ' + METER.r + ' ' + METER.r + ' 0 0 1 ' +
               p2.x.toFixed(2) + ' ' + p2.y.toFixed(2);
    }

    function buildMeter() {
        if (!meterBandsEl) return;

        var n = METER_SEGMENTS.length;
        // Segments touch — colors form a single ice→fire gradient
        var gap = 0;

        // Create a parallel <g> for the small in-segment numerals if not present
        var numeralsEl = document.getElementById('meter-numerals');
        if (!numeralsEl) {
            numeralsEl = document.createElementNS(SVG_NS, 'g');
            numeralsEl.setAttribute('id', 'meter-numerals');
            numeralsEl.setAttribute('class', 'meter-numerals');
            // insert before the needle so the needle renders on top
            if (meterNeedleEl && meterNeedleEl.parentNode) {
                meterNeedleEl.parentNode.insertBefore(numeralsEl, meterNeedleEl);
            }
        }

        METER_SEGMENTS.forEach(function(seg, i) {
            var t1 = (i / n) + gap;
            var t2 = ((i + 1) / n) - gap;
            var path = document.createElementNS(SVG_NS, 'path');
            path.setAttribute('class', 'band band-' + seg.id);
            path.setAttribute('d', arcPathT(t1, t2));
            meterBandsEl.appendChild(path);

            // Numeral inside the segment, on the arc
            var tc = (i + 0.5) / n;
            var pos = pointOnArcT(tc, METER.r);
            var text = document.createElementNS(SVG_NS, 'text');
            text.setAttribute('x', pos.x.toFixed(1));
            text.setAttribute('y', pos.y.toFixed(1));
            text.setAttribute('text-anchor', 'middle');
            text.setAttribute('dominant-baseline', 'middle');
            text.textContent = seg.numeral;
            numeralsEl.appendChild(text);
        });
    }

    function levelToSegmentIndex(system, level, allClear) {
        if (allClear || !system || system === 'none') return 5; // COMFORT (center)
        var n = Number(level);
        if (!isFinite(n)) return 5;
        if (system === 'ccf') {
            // CCF1..CCF5 → segments 0..4
            return Math.max(0, Math.min(4, n - 1));
        }
        if (system === 'juicecon') {
            // JC5..JC1 → segments 6..10
            return Math.max(6, Math.min(10, 11 - n));
        }
        return 5;
    }

    function setNeedleToLevel(system, level, allClear) {
        if (!meterNeedleEl) return;
        var idx = levelToSegmentIndex(system, level, allClear);
        var t = (idx + 0.5) / METER_SEGMENTS.length;
        var deg = -90 + t * 180;
        meterNeedleEl.style.transform = 'rotate(' + deg.toFixed(2) + 'deg)';
    }

    function bindEvents() {
        refreshBtn.addEventListener('click', manualRefresh);
        locationBtn.addEventListener('click', function() {
            openModal(modalEl, locationBtn);
        });
        retryBtn.addEventListener('click', manualRefresh);
        modalCancel.addEventListener('click', function() {
            closeModal(modalEl);
        });
        modalSubmit.addEventListener('click', submitZip);
        zipInput.addEventListener('keypress', function(e) {
            if (e.key === 'Enter') submitZip();
        });
        infoBtn.addEventListener('click', function() {
            openModal(infoModal, infoBtn);
        });
        infoClose.addEventListener('click', function() {
            closeModal(infoModal);
        });
        shareBtn.addEventListener('click', shareDewcon);

        // Global Escape key handler for modals
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                if (!infoModal.classList.contains('hidden')) {
                    closeModal(infoModal);
                } else if (!modalEl.classList.contains('hidden')) {
                    closeModal(modalEl);
                }
            }
        });
    }

    // =====================
    // Modal focus management
    // =====================

    function getFocusableElements(container) {
        var elements = container.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        return Array.prototype.filter.call(elements, function(el) {
            return !el.disabled && el.offsetParent !== null;
        });
    }

    function openModal(modal, triggerEl) {
        modalTriggerElement = triggerEl || document.activeElement;
        modal.classList.remove('hidden');

        // If this is the location modal, also hide loading
        if (modal === modalEl) {
            loadingEl.classList.add('hidden');
            zipInput.value = '';
        }

        // Focus first focusable element
        var focusable = getFocusableElements(modal);
        if (modal === modalEl && zipInput) {
            zipInput.focus();
        } else if (focusable.length > 0) {
            focusable[0].focus();
        }

        // Attach focus trap
        modal._trapHandler = function(e) {
            trapFocus(e, modal);
        };
        modal.addEventListener('keydown', modal._trapHandler);
    }

    function closeModal(modal) {
        modal.classList.add('hidden');

        // Remove focus trap
        if (modal._trapHandler) {
            modal.removeEventListener('keydown', modal._trapHandler);
            modal._trapHandler = null;
        }

        // Restore focus to trigger element
        if (modalTriggerElement && typeof modalTriggerElement.focus === 'function') {
            modalTriggerElement.focus();
            modalTriggerElement = null;
        }

        // If location modal closed with no location, show error
        if (modal === modalEl && currentLat === null && currentZip === null) {
            showError('Please enter a ZIP code to check DEWCON status');
        }
    }

    function trapFocus(e, modal) {
        if (e.key !== 'Tab') return;

        var focusable = getFocusableElements(modal);
        if (focusable.length === 0) return;

        var firstEl = focusable[0];
        var lastEl = focusable[focusable.length - 1];

        if (e.shiftKey) {
            // Shift+Tab: if on first element, wrap to last
            if (document.activeElement === firstEl) {
                e.preventDefault();
                lastEl.focus();
            }
        } else {
            // Tab: if on last element, wrap to first
            if (document.activeElement === lastEl) {
                e.preventDefault();
                firstEl.focus();
            }
        }
    }

    // =====================
    // Auto-refresh
    // =====================

    function startAutoRefresh() {
        stopAutoRefresh();
        autoRefreshTimer = setInterval(function() {
            if (currentZip || currentLat !== null) {
                fetchData();
            }
        }, AUTO_REFRESH_INTERVAL);
        updateRefreshStatus();
    }

    function stopAutoRefresh() {
        if (autoRefreshTimer) {
            clearInterval(autoRefreshTimer);
            autoRefreshTimer = null;
        }
    }

    function updateRefreshStatus() {
        if (!lastRefreshTime || !refreshStatusEl) return;

        var timeStr = lastRefreshTime.toLocaleTimeString('en-US', {
            hour: 'numeric',
            minute: '2-digit',
            hour12: true
        });
        refreshStatusTextEl.textContent = 'LAST UPDATED ' + timeStr + ' // NEXT IN 15 MIN';
    }

    function manualRefresh() {
        // Reset the auto-refresh timer on manual refresh
        if (currentZip || currentLat !== null) {
            refresh();
        } else {
            getLocation();
        }
    }

    // Location
    function getLocation() {
        showLoading();

        if (!navigator.geolocation) {
            openModal(modalEl, null);
            return;
        }

        navigator.geolocation.getCurrentPosition(
            function(position) {
                currentLat = position.coords.latitude;
                currentLon = position.coords.longitude;
                currentZip = null;
                fetchData();
            },
            function(error) {
                console.log('Geolocation error:', error);
                openModal(modalEl, null);
            },
            { timeout: 10000 }
        );
    }

    // Test mode API call -- bypasses geolocation, sends dewpoint directly
    function fetchTestDewpoint(dp) {
        showLoading();
        fetch('/api/dewcon?dewpoint=' + encodeURIComponent(dp))
            .then(function(response) {
                return response.json().then(function(data) {
                    if (!response.ok) {
                        throw new Error(data.error || 'Unknown error');
                    }
                    return data;
                });
            })
            .then(function(data) {
                updateDisplay(data);
            })
            .catch(function(error) {
                showError(error.message);
            });
    }

    // API
    function fetchData() {
        showLoading();

        var url = '/api/dewcon?';
        if (currentZip) {
            url += 'zip=' + encodeURIComponent(currentZip);
        } else if (currentLat !== null && currentLon !== null) {
            url += 'lat=' + currentLat + '&lon=' + currentLon;
        } else {
            showError('No location available');
            return;
        }

        fetch(url)
            .then(function(response) {
                return response.json().then(function(data) {
                    if (!response.ok) {
                        throw new Error(data.error || 'Unknown error');
                    }
                    return data;
                });
            })
            .then(function(data) {
                updateDisplay(data);
            })
            .catch(function(error) {
                showError(error.message);
            });
    }

    // Map NWS textDescription + cloud cover + (fallback) temp/dewpoint to a
    // discrete weather token used by the CSS background layers. Tokens:
    //   clear | partly | cloudy | rain | storm | snow | fog
    function classifyWeather(data) {
        var cond = (data.condition || '').toLowerCase();
        var w = null;

        if (cond) {
            if (/thunder|t-?storm|tstm|squall/.test(cond))           w = 'storm';
            else if (/snow|sleet|flurr|ice pellet|wintry/.test(cond)) w = 'snow';
            else if (/rain|drizzle|shower/.test(cond))                w = 'rain';
            else if (/fog|mist|haze|smoke/.test(cond))                w = 'fog';
            else if (/overcast/.test(cond))                           w = 'cloudy';
            else if (/cloud/.test(cond)) {
                w = /partly|few|mostly clear|scattered/.test(cond) ? 'partly' : 'cloudy';
            } else if (/clear|fair|sunny/.test(cond))                 w = 'clear';
        }

        if (!w && typeof data.cloudCoverPct === 'number') {
            var c = data.cloudCoverPct;
            if (c <= 15)      w = 'clear';
            else if (c <= 55) w = 'partly';
            else if (c <= 85) w = 'cloudy';
            else              w = 'cloudy';
        }

        if (!w) {
            // Last-resort heuristic from temp + dewpoint
            var t = data.temperature;
            var d = data.dewpoint;
            if (t != null && d != null) {
                var spread = t - d;
                if (t <= 32 && spread < 6)      w = 'snow';
                else if (spread < 4)            w = 'fog';
                else if (d >= 65 && spread < 8) w = 'cloudy';
                else                            w = 'clear';
            } else {
                w = 'clear';
            }
        }

        var tod = data.isDaytime === false ? 'night' : 'day';
        return { weather: w, tod: tod };
    }

    // Display
    function updateDisplay(data) {
        currentData = data;
        var system = data.activeSystem || 'none';
        var level = data.allClear ? 'clear' : data.level;

        // Set system and level on body for CSS color switching
        document.body.setAttribute('data-system', system);
        document.body.setAttribute('data-level', level);

        // Set weather + time-of-day on body for sky background layers
        var w = classifyWeather(data);
        document.body.setAttribute('data-weather', w.weather);
        document.body.setAttribute('data-tod', w.tod);

        // Update meter chip (compact level callout below the needle pivot)
        if (data.allClear) {
            meterChipEl.textContent = 'ALL CLEAR';
            meterChipEl.classList.add('all-clear');
        } else {
            var sysName = data.systemName || 'DEWCON';
            meterChipEl.textContent = sysName + ' ' + data.level;
            meterChipEl.classList.remove('all-clear');
        }

        // Drive the needle (categorical, points at active level) and the live dewpoint readout
        meterReadoutValueEl.textContent = data.dewpoint.toFixed(1) + '°F';
        setNeedleToLevel(system, data.level, data.allClear);

        // Update text
        descriptorEl.textContent = data.descriptor.toUpperCase();
        descriptionEl.textContent = '"' + data.description + '"';

        // Update data panel
        var protocolName = systemNames[system] || 'DEWCON';
        protocolEl.innerHTML = '<strong></strong>';
        protocolEl.firstChild.textContent = protocolName;
        temperatureEl.textContent = data.temperature != null ? data.temperature.toFixed(1) + '\u00B0F' : 'N/A';
        dewpointEl.textContent = data.dewpoint.toFixed(1) + '\u00B0F';
        locationEl.textContent = data.location.city + ', ' + data.location.state;
        stationEl.textContent = data.location.station;
        timestampEl.textContent = formatTimestamp(data.timestamp);

        // Track refresh time and start auto-refresh
        lastRefreshTime = new Date();
        startAutoRefresh();

        showDisplay();
    }

    function formatTimestamp(isoString) {
        try {
            var date = new Date(isoString);
            return date.toLocaleTimeString('en-US', {
                hour: 'numeric',
                minute: '2-digit',
                hour12: true,
                timeZoneName: 'short'
            });
        } catch (e) {
            return '--:-- --';
        }
    }

    // Share
    function shareDewcon() {
        if (!currentData) return;

        var data = currentData;
        var system = data.activeSystem || 'none';
        var levelText;
        if (data.allClear) {
            levelText = 'DEWCON ALL CLEAR';
        } else {
            var prefix = data.systemName || 'DEWCON';
            levelText = prefix + ' ' + data.level;
        }

        var locationText = data.location.city + ', ' + data.location.state;
        var shareText = levelText + ' in ' + locationText + ' // Dewpoint: ' + data.dewpoint.toFixed(1) + '\u00B0F // ' + data.descriptor;
        var shareUrl = 'https://juicecon.fly.dev';

        if (navigator.share) {
            navigator.share({
                title: 'DEWCON - Atmospheric Moisture Assessment',
                text: shareText,
                url: shareUrl
            }).catch(function(err) {
                // User cancelled or share failed, ignore
            });
        } else {
            var clipboardText = shareText + ' ' + shareUrl;
            navigator.clipboard.writeText(clipboardText).then(function() {
                var original = shareBtn.textContent;
                shareBtn.textContent = 'COPIED';
                setTimeout(function() {
                    shareBtn.textContent = original;
                }, 2000);
            }).catch(function() {
                // Fallback for older browsers without clipboard API
                var textArea = document.createElement('textarea');
                textArea.value = clipboardText;
                textArea.style.position = 'fixed';
                textArea.style.opacity = '0';
                document.body.appendChild(textArea);
                textArea.select();
                document.execCommand('copy');
                document.body.removeChild(textArea);
                var original = shareBtn.textContent;
                shareBtn.textContent = 'COPIED';
                setTimeout(function() {
                    shareBtn.textContent = original;
                }, 2000);
            });
        }
    }

    // UI State
    function showLoading() {
        loadingEl.classList.remove('hidden');
        errorEl.classList.add('hidden');
        displayEl.classList.add('hidden');
    }

    function showError(message) {
        loadingEl.classList.add('hidden');
        errorEl.classList.remove('hidden');
        displayEl.classList.add('hidden');
        errorMessageEl.textContent = message;
    }

    function showDisplay() {
        loadingEl.classList.add('hidden');
        errorEl.classList.add('hidden');
        displayEl.classList.remove('hidden');
    }

    // Modal (legacy wrappers kept for submitZip flow)
    function showModal() {
        openModal(modalEl, null);
    }

    function hideModal() {
        closeModal(modalEl);
    }

    function submitZip() {
        var zip = zipInput.value.trim();
        if (!/^\d{5}$/.test(zip)) {
            zipInput.style.borderColor = 'var(--oxblood-text)';
            return;
        }

        currentZip = zip;
        currentLat = null;
        currentLon = null;
        closeModal(modalEl);
        fetchData();
    }

    function refresh() {
        if (currentZip) {
            fetchData();
        } else if (currentLat !== null) {
            fetchData();
        } else {
            getLocation();
        }
    }

    // Start
    init();
})();
