(function () {
    "use strict";

    var CFG = window.__DASHBOARD_CHARTS__ || {};
    var PALETTE = {
        teal: "#14b8b8",
        blue: "#3b82f6",
        orange: "#f97316",
        green: "#10b981",
        red: "#ef4444",
        purple: "#8b5cf6",
        indigo: "#6366f1",
        yellow: "#eab308",
        slate: "#94a3b8"
    };

    function inr(v) {
        return "₹" + Number(v || 0).toLocaleString("en-IN", { maximumFractionDigits: 0 });
    }

    function hasData(list) {
        return list && list.length > 0;
    }

    function initChart(canvasId, emptyId, build) {
        var canvas = document.getElementById(canvasId);
        if (!canvas) return;
        var data = build();
        if (!data) {
            canvas.style.display = "none";
            var empty = document.getElementById(emptyId);
            if (empty) empty.style.display = "flex";
            return;
        }
        new Chart(canvas, data);
    }

    function initRevenue() {
        initChart("chart-revenue", "chart-revenue-empty", function () {
            if (!hasData(CFG.revenueByDay)) return null;
            return {
                type: "line",
                data: {
                    labels: CFG.revenueByDay.map(function (d) { return d.Day ? d.Day.slice(5) : ""; }),
                    datasets: [{
                        label: "Revenue",
                        data: CFG.revenueByDay.map(function (d) { return d.Total; }),
                        borderColor: PALETTE.teal,
                        backgroundColor: "rgba(20, 184, 184, 0.12)",
                        fill: true,
                        tension: 0.35,
                        borderWidth: 2,
                        pointRadius: 2,
                        pointHoverRadius: 5
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            callbacks: {
                                label: function (ctx) { return " " + inr(ctx.parsed.y); }
                            }
                        }
                    },
                    scales: {
                        x: { ticks: { maxTicksLimit: 8, font: { size: 10 } }, grid: { display: false } },
                        y: {
                            ticks: { font: { size: 10 }, callback: function (v) { return inr(v); } },
                            grid: { color: "rgba(148, 163, 184, 0.15)" }
                        }
                    }
                }
            };
        });
    }

    function initStatus() {
        initChart("chart-status", "chart-status-empty", function () {
            var counts = CFG.statusCounts || {};
            var entries = Object.keys(counts).filter(function (k) { return counts[k] > 0; });
            if (entries.length === 0) return null;
            var colors = {
                scheduled: PALETTE.purple,
                assigned: PALETTE.indigo,
                started: PALETTE.orange,
                reached_pickup: PALETTE.blue,
                in_transit: PALETTE.teal,
                completed: PALETTE.green,
                cancelled: PALETTE.red,
                draft: PALETTE.slate
            };
            return {
                type: "doughnut",
                data: {
                    labels: entries.map(function (k) { return k.replace(/_/g, " "); }),
                    datasets: [{
                        data: entries.map(function (k) { return counts[k]; }),
                        backgroundColor: entries.map(function (k) { return colors[k] || PALETTE.slate; }),
                        borderWidth: 2,
                        borderColor: "#ffffff"
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    cutout: "62%",
                    plugins: {
                        legend: {
                            position: "bottom",
                            labels: { boxWidth: 10, boxHeight: 10, font: { size: 10 } }
                        }
                    }
                }
            };
        });
    }

    function initBookings() {
        initChart("chart-bookings", "chart-bookings-empty", function () {
            if (!hasData(CFG.bookingsByDay)) return null;
            return {
                type: "bar",
                data: {
                    labels: CFG.bookingsByDay.map(function (d) { return d.Day ? d.Day.slice(5) : ""; }),
                    datasets: [{
                        label: "Bookings",
                        data: CFG.bookingsByDay.map(function (d) { return d.Count; }),
                        backgroundColor: "rgba(59, 130, 246, 0.75)",
                        hoverBackgroundColor: PALETTE.blue,
                        borderRadius: 4
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: { display: false },
                        tooltip: {
                            callbacks: {
                                label: function (ctx) { return " " + ctx.parsed.y + " bookings"; }
                            }
                        }
                    },
                    scales: {
                        x: { ticks: { maxTicksLimit: 8, font: { size: 10 } }, grid: { display: false } },
                        y: { beginAtZero: true, ticks: { font: { size: 10 }, precision: 0 }, grid: { color: "rgba(148, 163, 184, 0.15)" } }
                    }
                }
            };
        });
    }

    function trackClick(target) {
        if (!window.fetch || !CFG.variant) return;
        var body = {
            experiment: "dashboard_v2",
            variant: CFG.variant,
            event: "dashboard_click",
            meta: { target: target }
        };
        fetch("/dashboard/event", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
            keepalive: true
        }).catch(function () {});
    }

    document.addEventListener("click", function (e) {
        var el = e.target.closest ? e.target.closest("[data-drill]") : null;
        if (!el) return;
        trackClick(el.getAttribute("data-drill") || "row");
    });

    function boot() {
        if (typeof Chart === "undefined") return;
        initRevenue();
        initStatus();
        initBookings();
    }

    if (document.readyState === "loading") {
        document.addEventListener("DOMContentLoaded", boot);
    } else {
        boot();
    }
})();