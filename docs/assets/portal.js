/* ==========================================================================
   EdgeOS Docs — portal runtime
   零依赖 · 无外部请求
   覆盖：主题（持久化 + 跟随系统）· 代码高亮与复制 · 首页四大交互区 ·
        控制平面抽屉 · 双总线模拟器 · 六工步流水线 · 点位规范库 ·
        全局命令面板 · Toast · 亚毫秒遥测时钟 · Hero 星座视觉
   ========================================================================== */

(function () {
  'use strict';

  var THEME_KEY = 'edgeos-docs-theme';
  var root = document.documentElement;

  /* ======================================================================
     0. 工具
     ====================================================================== */

  function $(sel, ctx) {
    return (ctx || document).querySelector(sel);
  }

  function $$(sel, ctx) {
    return Array.prototype.slice.call((ctx || document).querySelectorAll(sel));
  }

  function escapeHtml(str) {
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }

  function escapeAttr(str) {
    return escapeHtml(str).replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }

  function nowTime() {
    var d = new Date();
    return [d.getHours(), d.getMinutes(), d.getSeconds()]
      .map(function (n) { return String(n).padStart(2, '0'); })
      .join(':');
  }

  function supportsMatchMedia() {
    return typeof window.matchMedia === 'function';
  }

  function matchMedia(query) {
    return supportsMatchMedia() ? window.matchMedia(query) : null;
  }

  function prefersReducedMotion() {
    var mq = matchMedia('(prefers-reduced-motion: reduce)');
    return !!(mq && mq.matches);
  }

  /* ======================================================================
     1. 主题：持久化 + 首次跟随系统
     ====================================================================== */

  function applyTheme(theme, source) {
    root.setAttribute('data-theme', theme);
    root.dataset.themeSource = source;
    var meta = $('meta[name="theme-color"]');
    if (meta) meta.setAttribute('content', theme === 'dark' ? '#000000' : '#F5F5F7');
  }

  function initTheme() {
    var btn = $('[data-theme-toggle]');
    var current = root.getAttribute('data-theme') || 'light';
    applyTheme(current, root.dataset.themeSource || 'system');

    if (btn) {
      btn.addEventListener('click', function () {
        var next = (root.getAttribute('data-theme') === 'dark') ? 'light' : 'dark';
        applyTheme(next, 'manual');
        try { localStorage.setItem(THEME_KEY, next); } catch (e) {}
        Toast.show(next === 'dark' ? '已切换至深色冷轧钛合金机载模式' : '已切换至亮色光学微晶玻璃模式', 'info');
      });
    }

    // 用户未手动指定时，实时跟随系统外观
    var mq = matchMedia('(prefers-color-scheme: dark)');
    if (!mq) return;

    var onChange = function (e) {
      if (root.dataset.themeSource === 'manual') return;
      applyTheme(e.matches ? 'dark' : 'light', 'system');
    };
    if (typeof mq.addEventListener === 'function') mq.addEventListener('change', onChange);
    else if (typeof mq.addListener === 'function') mq.addListener(onChange);
  }

  /* ======================================================================
     2. Toast
     ====================================================================== */

  var Toast = (function () {
    var el = null;
    var timer = null;

    function ensure() {
      if (el) return el;
      el = $('#toast');
      return el;
    }

    function show(text, variant) {
      var node = ensure();
      if (!node) return;
      node.dataset.variant = variant || 'success';
      var textEl = node.querySelector('.toast__text');
      if (textEl) textEl.textContent = text;
      node.classList.add('is-visible');
      if (timer) clearTimeout(timer);
      timer = setTimeout(function () {
        node.classList.remove('is-visible');
      }, 2600);
    }

    return { show: show };
  })();

  function showToast(text, variant) {
    Toast.show(text, variant);
  }

  /* ======================================================================
     3. 剪贴板
     ====================================================================== */

  function copyText(text) {
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).catch(function () { legacyCopy(text); });
      return;
    }
    legacyCopy(text);
  }

  function legacyCopy(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); } catch (e) {}
    document.body.removeChild(ta);
  }

  /* ======================================================================
     4. 文档内页：代码高亮 + 复制按钮
     ====================================================================== */

  var KEYWORDS = [
    'package', 'import', 'func', 'return', 'type', 'struct', 'interface', 'var', 'const', 'let',
    'if', 'else', 'for', 'range', 'switch', 'case', 'break', 'continue', 'go', 'defer', 'map',
    'true', 'false', 'null', 'nil', 'undefined', 'async', 'await', 'export', 'default', 'from',
    'SELECT', 'FROM', 'WHERE', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'TABLE', 'AND', 'OR'
  ];

  function highlightCode() {
    var blocks = $$('.doc-content pre > code');
    blocks.forEach(function (code) {
      // 已被 rouge 等高亮器处理过则跳过，避免二次包裹
      if (code.querySelector('span')) return;
      if (code.dataset.portalHighlighted === '1') return;

      var lang = '';
      var cls = code.className || '';
      // 语言标识可能落在 <code> 自身，也可能落在外层
      // <div class="language-json highlighter-rouge">（kramdown + rouge 的默认产物）
      var host = code.closest ? code.closest('[class*="language-"]') : null;
      if (host && host !== code) cls += ' ' + host.className;
      var m = cls.match(/language-([\w-]+)/);
      if (m) lang = m[1].toLowerCase();

      var raw = code.textContent;
      if (!raw || raw.length > 60000) return;

      code.dataset.portalHighlighted = '1';
      code.innerHTML = tokenize(raw, lang);
    });
  }

  function tokenize(src, lang) {
    var hashComment = ['bash', 'sh', 'shell', 'yaml', 'yml', 'toml', 'ini', 'python', 'py', 'conf'].indexOf(lang) >= 0;

    // 必须使用命名分组：# 注释是按语言条件拼接的，若用编号分组（mt[1]/mt[2]…）
    // 会让非 hash 注释语言（json / go / js / c）的分组索引整体前移一位，
    // 结果字符串被染成注释色、关键字被染成字符串色、数字被染成关键字色。
    var pattern = new RegExp(
      '(?<comment>\\/\\*[\\s\\S]*?\\*\\/|\\/\\/[^\\n]*' + (hashComment ? '|#[^\\n]*' : '') + ')' +
      '|(?<string>"(?:[^"\\\\\\n]|\\\\.)*"|\'(?:[^\'\\\\\\n]|\\\\.)*\')' +
      '|\\b(?<keyword>' + KEYWORDS.join('|') + ')\\b' +
      '|\\b(?<number>\\d+(?:\\.\\d+)?)\\b',
      'g'
    );

    var out = '';
    var last = 0;
    var mt;

    while ((mt = pattern.exec(src)) !== null) {
      out += escapeHtml(src.slice(last, mt.index));
      var g = mt.groups || {};
      if (g.comment !== undefined) {
        out += '<span class="tok-comment">' + escapeHtml(mt[0]) + '</span>';
      } else if (g.string !== undefined) {
        out += '<span class="tok-string">' + escapeHtml(mt[0]) + '</span>';
      } else if (g.keyword !== undefined) {
        out += '<span class="tok-keyword">' + escapeHtml(mt[0]) + '</span>';
      } else if (g.number !== undefined) {
        out += '<span class="tok-number">' + escapeHtml(mt[0]) + '</span>';
      }
      last = mt.index + mt[0].length;
      if (mt[0].length === 0) pattern.lastIndex++;
    }
    out += escapeHtml(src.slice(last));
    return out;
  }

  function addCopyButtons() {
    $$('.doc-content pre').forEach(function (pre) {
      if (pre.querySelector('.copy-btn')) return;
      var code = pre.querySelector('code');
      if (!code) return;

      var btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'copy-btn';
      btn.textContent = '复制';
      btn.setAttribute('aria-label', '复制代码');
      btn.addEventListener('click', function () {
        copyText(code.textContent);
        btn.textContent = '已复制';
        btn.classList.add('is-done');
        setTimeout(function () {
          btn.textContent = '复制';
          btn.classList.remove('is-done');
        }, 1600);
      });
      pre.appendChild(btn);
    });
  }

  /* ======================================================================
     5. Tab 切换（首页四大交互区）
     ====================================================================== */

  var TAB_KEYS = ['explorer', 'bus', 'pipeline', 'specs'];

  /* 滚动型分区：不切换工作台面板，只负责定位与高亮（顶部胶囊与中部胶囊共用） */
  var SCROLL_TABS = {
    overview: '#overview',
    scenarios: '#scenarios',
    visualization: '#visualization',
    docs: '#documents'
  };

  /* 中部工作台标题随分段器联动 */
  var VIEW_META = {
    explorer: {
      eyebrow: 'CORE SYSTEM ARCHITECTURE',
      title: '六大核心控制平面',
      note: '点击任一平面，即可展开对应的 EAN 2.0 真实报文与关键指标。'
    },
    bus: {
      eyebrow: 'SYMMETRIC DUAL PROTOCOL BUS',
      title: '双通道对称传输',
      note: 'MQTT 与 NATS 采用完全同构的寻址规范，切换载波时分隔符自动改写。'
    },
    pipeline: {
      eyebrow: 'DETERMINISTIC WORKFLOW',
      title: '流水线工步执行器',
      note: '从端侧接入到工程终审，全链路受确定性安全状态机约束。'
    },
    specs: {
      eyebrow: 'ENGINEERING SPECIFICATION',
      title: '寄存器点位与 Capability 映射',
      note: 'Modbus 保持寄存器占用 40001+ 编号；报文中的 address 为 PDU 偏移（0 基）。'
    }
  };

  /* 统一刷新分段控制器高亮（桌面胶囊 + 移动端导航条 + 滚动型按钮） */
  function setActiveNav(key) {
    $$('[data-nav-tab]').forEach(function (btn) {
      btn.classList.toggle('is-active', btn.dataset.navTab === key);
    });
    $$('[data-nav-scroll]').forEach(function (btn) {
      btn.classList.toggle('is-active', SCROLL_TABS[key] === btn.dataset.navScroll);
    });
  }

  function scrollToEl(sel) {
    var el = $(sel);
    if (!el) return;
    if (prefersReducedMotion()) el.scrollIntoView({ block: 'start' });
    else el.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }

  function switchTab(key, options) {
    var opts = options || {};

    /* 滚动型分段（系统总览 / 场景用例 / 文档）：只做导航定位，不改变工作台面板 */
    if (SCROLL_TABS[key]) {
      setActiveNav(key);
      if (!opts.skipScroll) scrollToEl(SCROLL_TABS[key]);
      return;
    }

    TAB_KEYS.forEach(function (k) {
      var panel = $('#panel-' + k);
      if (!panel) return;
      var isTarget = k === key;
      panel.hidden = !isTarget;
      panel.classList.toggle('is-visible', isTarget);
    });

    setActiveNav(key);

    /* 标题联动 */
    var meta = VIEW_META[key];
    if (meta) {
      var eyebrow = $('#view-eyebrow');
      var title = $('#view-title');
      var note = $('#view-note');
      if (eyebrow) eyebrow.textContent = meta.eyebrow;
      if (title) title.textContent = meta.title;
      if (note) note.textContent = meta.note;
    }

    if (opts.skipScroll) return;
    scrollToEl('#panel-' + key);
  }

  /* ======================================================================
     6. 首页数据：六大控制平面（真实 EAN 2.0 报文）
     ====================================================================== */

  var SUBSYSTEMS = [
    {
      code: '01 / DISCOVERY',
      short: '自动发现',
      title: '自动发现节点',
      sub: 'Discovery Center',
      latency: '< 30s 心跳窗口',
      brief: '节点上线即自动注册，无需现场硬编码 IP 地址。',
      glyph:
        '<circle cx="12" cy="12" r="1.6" fill="currentColor" stroke="none"/>' +
        '<path d="M8.6 8.6a4.8 4.8 0 000 6.8"/>' +
        '<path d="M15.4 8.6a4.8 4.8 0 010 6.8"/>' +
        '<path d="M5.8 5.8a8.8 8.8 0 000 12.4"/>' +
        '<path d="M18.2 5.8a8.8 8.8 0 010 12.4"/>',
      desc: 'edgeCore 启动后主动向 $edgeos/discovery/agent 发布 Agent Descriptor，并单独发布 Capability Descriptor。EdgeOS 汇聚为统一 Registry，无需现场硬编码 IP。',
      slo: [
        { k: '发现信道', v: '$edgeos/discovery/agent', amber: false },
        { k: 'Capability 上报', v: '$edgeos/discovery/capability', amber: false },
        { k: '降级信道', v: 'mDNS 广播 + 主动 query/response', amber: true }
      ],
      payload: {
        header: {
          message_id: 'msg-agent-desc-001',
          timestamp: 1776787200000,
          source: 'edgeCore-node-001',
          message_type: 'agent_descriptor',
          version: '2.0'
        },
        body: {
          agent: {
            id: 'edgeCore-node-001',
            kind: 'device',
            version: '2.0.0',
            status: 'online',
            transport: 'mqtt',
            heartbeat_interval_sec: 30,
            endpoint: { host: '192.168.1.100', port: 8082 },
            metadata: { os: 'linux', arch: 'arm64', model: 'edgeCore-gateway-pro' }
          }
        }
      }
    },
    {
      code: '02 / ORCHESTRATION',
      short: '编排调度',
      title: '指令编排调度',
      sub: 'Invoke Orchestrator',
      latency: '默认超时 10s / retry 0~5',
      brief: '按 Capability 下发指令，全生命周期状态可追踪。',
      glyph:
        '<circle cx="6" cy="6" r="2.4"/>' +
        '<circle cx="6" cy="18" r="2.4"/>' +
        '<circle cx="18" cy="12" r="2.4"/>' +
        '<path d="M8.4 6H13a2 2 0 012 2v1.6"/>' +
        '<path d="M8.4 18H13a2 2 0 002-2v-1.6"/>',
      desc: 'EdgeOS 向 $edgeos/invoke/{agent_id} 下发 invoke_capability 信封，edgeCore 校验 target / Capability / 权限后执行，并回 $edgeos/reply/{source}。状态机 queued → running → completed / failed / timeout / rejected。',
      slo: [
        { k: '请求信道', v: '$edgeos/invoke/{agent_id}', amber: false },
        { k: '应答信道', v: '$edgeos/reply/{source_agent_id}', amber: false },
        { k: '失败回退', v: 'V1.0 通道 $edgeCore/cmd/{node} 重试', amber: true }
      ],
      payload: {
        header: {
          message_id: 'msg-invoke-001',
          timestamp: 1776787200000,
          source: 'edgeos-planner-001',
          destination: 'edgeCore-node-001',
          message_type: 'invoke_capability',
          version: '2.0',
          correlation_id: 'req-plan-001'
        },
        body: {
          invoke_id: 'invoke-001',
          target: 'edgeCore-node-001',
          capability: 'modbus_tcp.write_point',
          arguments: { device_id: 'slave-1', address: '40001', value: 25.5 },
          options: { timeout_sec: 10, priority: 'normal', retry: 2 }
        }
      }
    },
    {
      code: '03 / EVENT CENTER',
      short: '事件监听',
      title: '设备事件监听',
      sub: 'Event Center',
      latency: 'Shadow COW 写入即发',
      brief: '变更事件携带前值上报，支撑参数追溯回溯。',
      glyph:
        '<path d="M6 10a6 6 0 1112 0v4.2l1.6 2.6H4.4L6 14.2V10z"/>' +
        '<path d="M10 20.4a2.2 2.2 0 004 0"/>',
      desc: 'ShadowCore 写入时在通知克隆中附加变更前值 previous_value，经 $edgeos/event/{agent_id}/{device_id} 上报。EdgeOS 打高精时标落库，支撑工业追溯与参数回溯。',
      slo: [
        { k: '事件信道', v: '$edgeos/event/{agent_id}/{device_id}', amber: false },
        { k: '前值语义', v: 'previous_value（仅通知克隆，不落快照）', amber: false },
        { k: '断链缓冲', v: '本地 NVMe 溢出缓冲后补传', amber: true }
      ],
      payload: {
        event_type: 'temperature.changed',
        device_id: 'slave-1',
        point_id: 'temperature',
        value: 45.2,
        previous_value: 42.1,
        metadata: { quality: 'good', channel_id: 'ch-1' }
      }
    },
    {
      code: '04 / GOVERNANCE',
      short: '运行治理',
      title: '运行安全治理',
      sub: 'Registry & Governance',
      latency: 'heartbeat 30s / QoS 0',
      brief: '心跳探活叠加资源锁，杜绝点位并发抢占。',
      glyph:
        '<path d="M12 3.2l7 2.6v5.4c0 4.4-2.9 8.2-7 9.2-4.1-1-7-4.8-7-9.2V5.8l7-2.6z"/>' +
        '<path d="M9.2 12.2l2 2 3.6-3.8"/>',
      desc: '按 heartbeat_interval_sec 高频探活，节点超时即判定离线并重平衡控制拓扑。执行层强制资源锁，防止同设备 / 同通道并发冲突。',
      slo: [
        { k: '心跳信道', v: '$edgeos/heartbeat/{agent_id}', amber: false },
        { k: '调度优先级', v: 'critical > high > normal > low', amber: false },
        { k: '冲突防护', v: 'resource_locks: device / channel 作用域', amber: true }
      ],
      payload: {
        registry_entry: {
          agent_id: 'edgeCore-node-001',
          status: 'online',
          last_heartbeat: 1776787200000,
          capabilities_indexed: 63,
          permission: 'readwrite'
        },
        options: {
          priority: 'critical',
          queue: 'priority',
          exclusive: true,
          resource_locks: [
            { resource: 'plc-001', scope: 'device' },
            { resource: 'com3', scope: 'channel' }
          ]
        }
      }
    },
    {
      code: '05 / DUAL CHANNEL',
      short: '双通道',
      title: '双通道对称传输',
      sub: 'MQTT 3.1.1 + NATS JetStream',
      latency: 'QoS 1 / 至少一次',
      brief: '两套总线同构寻址，只差一个分隔符。',
      glyph:
        '<path d="M4 8.5h13"/>' +
        '<path d="M14 4.8l3.7 3.7-3.7 3.7"/>' +
        '<path d="M20 15.5H7"/>' +
        '<path d="M10 11.8l-3.7 3.7 3.7 3.7"/>',
      desc: '两套总线并存互通，同一物理语义只差一个分隔符：MQTT 斜杠 Topic 与 NATS 点分 Subject 一一对应，业务层无需分支。通配符同步映射 + → * 、# → >。',
      slo: [
        { k: 'MQTT 面', v: '$edgeos/invoke/{agent_id} · 斜杠', amber: false },
        { k: 'NATS 面', v: '$edgeos.invoke.{agent_id} · 点分', amber: false },
        { k: '通配符', v: '+ → *   # → >', amber: true }
      ],
      payload: {
        mqtt_topic: '$edgeos/invoke/edgeCore-node-001',
        nats_subject: '$edgeos.invoke.edgeCore-node-001',
        wildcard: { mqtt: '$edgeos/event/+/+', nats: '$edgeos.event.*.*' },
        qos: 1,
        delivery: 'at_least_once'
      }
    },
    {
      code: '06 / COMPAT SHIM',
      short: '旧版兼容',
      title: '旧版协议热兼容',
      sub: 'V1.0 Compatibility Shim',
      latency: '并行双轨 · 零停机升级',
      brief: 'V1 通道并行保留，升级可分批次推进。',
      glyph:
        '<path d="M12 3.4l8.4 4.3L12 12.4 3.6 7.7 12 3.4z"/>' +
        '<path d="M3.6 12.2L12 16.5l8.4-4.3"/>' +
        '<path d="M3.6 16.6L12 20.9l8.4-4.3"/>',
      desc: '未升级 EAN 2.0 的旧 EdgeOS 继续使用 V1 通道。新功能一律走 $edgeos/*，旧链路不改动、互不干涉，升级可分批推进。',
      slo: [
        { k: 'V1 发现 / 心跳', v: 'edgeCore/nodes/register · edgeCore/heartbeat/{node}', amber: true },
        { k: 'V1 设备上报', v: 'edgeCore/devices/report', amber: true },
        { k: 'V1 写入', v: 'edgeCore/cmd/{node}/{device}/write', amber: true }
      ],
      payload: {
        v1_mqtt: {
          register: 'edgeCore/nodes/register',
          device_report: 'edgeCore/devices/report',
          heartbeat: 'edgeCore/heartbeat/{node}',
          write: 'edgeCore/cmd/{node}/{device}/write'
        },
        v1_nats: {
          register: 'edgeCore.nodes.register',
          device_report: 'edgeCore.devices.report',
          heartbeat: 'edgeCore.heartbeat.{node}',
          write: 'edgeCore.cmd.{node}.{device}.write'
        },
        coexistence: 'EAN 与 V1 并行，新功能仅走 $edgeos/*'
      }
    }
  ];

  /* ======================================================================
     7. 首页数据：六工步流水线
     ====================================================================== */

  var PIPELINE = [
    {
      badge: 'STEP 01',
      title: '自动发现节点 (Discovery)',
      text: '广播 discovery/query，在线节点回 Agent 与 Capability 描述符，EdgeOS 汇入 Registry。',
      code: '[INIT] publish $edgeos/discovery/query\n[RECV] agent_descriptor <- edgeCore-node-001 (arm64)\n[REGISTRY] capability_descriptor: 63 条已入册 · node -> online'
    },
    {
      badge: 'STEP 02',
      title: '指令编排调度 (Invoke)',
      text: '按 Capability 打包 invoke_capability 信封，经对称双总线派发并跟踪 invoke_id 全生命周期。',
      code: '[DISPATCH] $edgeos/invoke/edgeCore-node-001\n[CAPABILITY] modbus_tcp.write_point (timeout=10s retry=2)\n[REPLY] status=completed · latency_ms=120'
    },
    {
      badge: 'STEP 03',
      title: '实时事件监听 (Event)',
      text: '接收 Shadow 变更事件，保留 previous_value 前值语义，打高精时标写入时序存储。',
      code: '[EVENT] $edgeos/event/edgeCore-node-001/slave-1\n[TYPE] temperature.changed · 42.1 -> 45.2 (quality=good)\n[STORE] ts=1776787200000 已落库，previous_value 保留'
    },
    {
      badge: 'STEP 04',
      title: '集群治理监控 (Governance)',
      text: '按心跳周期探活，超时判定离线并重平衡拓扑；按 priority / resource_locks 串行化冲突操作。',
      code: '[HEALTH] $edgeos/heartbeat/edgeCore-node-001 (30s)\n[LOCK] resource=plc-001 scope=device granted\n[STATUS] capability_digest 未漂移 · governance OK'
    },
    {
      badge: 'STEP 05',
      title: '旧版协议兼容 (V1 Shim)',
      text: '旧 EdgeOS 继续走 V1 通道，新功能仅走 $edgeos/*，两套通道并行，升级可分批次推进。',
      code: '[LEGACY] ingest $edgeCore/devices/report\n[SHIM] map -> $edgeos/event/{agent_id}/{device_id}\n[STATUS] dual-track OK (V1 与 EAN 2.0 并行)'
    },
    {
      badge: 'STEP 06',
      title: '工程辅助与终审 (Human-in-the-loop)',
      text: 'AI 逆向与文档解析产出的一切配置均为草案，须经工程师签字后方可落库生效。',
      code: '[AI] capability=ai.protocol_reverse status=waiting_confirm\n[REVIEW] engineer sign-off required\n[APPLIED] 签字通过 · AI 草案已提升为正式配置'
    }
  ];

  /* ======================================================================
     7.1 首页数据：总线载荷预设
     把重复的报文手改工作收敛为四个一键预设，页面不再堆叠大段说明文字。
     ====================================================================== */

  function envelope(capability, args) {
    return {
      header: {
        message_id: 'msg-1001',
        source: 'edgeos-planner-001',
        destination: 'edgeCore-node-001',
        message_type: 'invoke_capability',
        version: '2.0'
      },
      body: {
        invoke_id: 'invoke-1001',
        target: 'edgeCore-node-001',
        capability: capability,
        arguments: args,
        options: { timeout_sec: 10, priority: 'normal', retry: 2 }
      }
    };
  }

  var BUS_PRESETS = [
    {
      key: 'write',
      label: '单点写入',
      hint: 'modbus_tcp.write_point',
      body: envelope('modbus_tcp.write_point', { device_id: 'slave-1', address: '40001', value: 25.5 })
    },
    {
      key: 'batch',
      label: '批量读取',
      hint: 'modbus_tcp.read_points',
      body: envelope('modbus_tcp.read_points', { device_id: 'slave-1', registers: ['40001', '40002', '40013'] })
    },
    {
      key: 'reverse',
      label: '协议逆向',
      hint: 'ai.protocol_reverse',
      body: envelope('ai.protocol_reverse', { device_id: 'unknown-plc', sample_capture: 'hex://a3f1c0', confirm_required: true })
    },
    {
      key: 'scan',
      label: '设备扫描',
      hint: 'device.scan',
      body: envelope('device.scan', { channel_id: 'ch-1', network: '192.168.3.0/24' })
    }
  ];

  /* ======================================================================
     7.2 首页数据：典型场景用例
     ====================================================================== */

  var SCENARIOS = [
    {
      no: '01',
      title: '多协议产线统一接入',
      desc: '新旧混线设备由 edgeCore 端侧归一，EdgeOS 统一编目、寻址与权限。',
      points: [
        'Modbus / S7 / DLT645 / BACnet / OPC UA 异构协议并存',
        '节点上线即自动注册 63 项 Capability，无需改动 PLC 程序'
      ],
      caps: ['Discovery', 'Capability Registry']
    },
    {
      no: '02',
      title: '跨工位节拍协同调度',
      desc: '多台 edgeCore 按同一节拍编排，主轴、进给与夹具跨节点同步动作。',
      points: [
        '统一 invoke_capability 信封，critical 优先级抢占队列',
        'resource_locks 串行化同设备写入，杜绝点位抢占'
      ],
      caps: ['Invoke Orchestrator', 'Resource Locks']
    },
    {
      no: '03',
      title: '无点表老设备 AI 逆向接入',
      desc: '现场缺少通信点表的存量设备，由 AI 解析手册与抓包产出配置草案。',
      points: [
        '协议逆向与文档解析结果一律标记为草案',
        '须经工程师在 Confirm API 签字后方可落库生效'
      ],
      caps: ['AI Co-pilot', 'Human Sign-off']
    },
    {
      no: '04',
      title: '链路抖动降级与断链补传',
      desc: '主通道超时自动重试，必要时切 V1.0 兼容通道，断链期间数据本地缓冲。',
      points: [
        'timeout_sec / retry 预算逐级降级，业务侧无感',
        'NVMe 溢出缓冲，链路恢复后按序补传前值事件'
      ],
      caps: ['Failover', 'Event Center']
    }
  ];

  /* ======================================================================
     7.3 首页数据：2.5D 可视化场景
     与产品内 ui/src/components/visual 同源：世界平面统一施加
     rotateX(60deg) rotateZ(-45deg)，每个立体物由「顶面 + 前面 + 侧面」三片
     沿 Z 轴拼合。这里用同一套几何在文档站静态复刻「产线展示」画面，
     坐标与 store/visual.ts 中的 M1~M5、AGV 完全一致。
     ====================================================================== */

  var ISO_STATUS = {
    running: {
      top: 'rgba(16,185,129,0.20)', front: 'rgba(16,185,129,0.40)',
      side: 'rgba(5,150,105,0.55)', color: '#10B981', label: '运行中',
      glow: false, foot: '设备运行正常'
    },
    standby: {
      top: 'rgba(148,163,184,0.20)', front: 'rgba(148,163,184,0.38)',
      side: 'rgba(100,116,139,0.52)', color: '#94A3B8', label: '待机',
      glow: false, foot: '设备运行正常'
    },
    warn: {
      top: 'rgba(245,158,11,0.22)', front: 'rgba(245,158,11,0.42)',
      side: 'rgba(180,120,10,0.55)', color: '#F59E0B', label: '关注',
      glow: true, foot: '设备状态需关注'
    },
    fault: {
      top: 'rgba(239,68,68,0.24)', front: 'rgba(239,68,68,0.44)',
      side: 'rgba(185,28,28,0.55)', color: '#EF4444', label: '故障',
      glow: true, foot: '设备状态需关注'
    }
  };

  var ISO_GRID = { cols: 10, rows: 7, cell: 48, scale: 0.94 };

  var ISO_MACHINES = [
    { id: 'M1', name: '1 号线 1 号机', col: 0, row: 2, status: 'running', oee: 92.4, rate: 46, temp: 58.2 },
    { id: 'M2', name: '1 号线 2 号机', col: 1, row: 2, status: 'running', oee: 91.8, rate: 45, temp: 61.5 },
    { id: 'M3', name: '1 号线 3 号机', col: 2, row: 2, status: 'warn', oee: 84.6, rate: 41, temp: 74.9 },
    { id: 'M4', name: '2 号线 1 号机', col: 6, row: 4, status: 'running', oee: 93.1, rate: 47, temp: 56.8 },
    { id: 'M5', name: '2 号线 2 号机', col: 7, row: 4, status: 'standby', oee: 78.2, rate: 33, temp: 42.1 }
  ];

  var ISO_AGVS = [
    { id: 'AGV-01', x: 130, y: 200, tx: 200, ty: 240, load: true, color: '#0EA5E9' },
    { id: 'AGV-02', x: 310, y: 140, tx: 360, ty: 120, load: false, color: '#8B5CF6' }
  ];

  /* 静态物：顶面 / 前面 / 侧面三档色阶 */
  function box(top, front, side) {
    return { top: top, front: front, side: side };
  }

  var ISO_STATIC = {
    wall: box('rgba(100,116,139,0.16)', 'rgba(100,116,139,0.26)', 'rgba(71,85,105,0.36)'),
    warehouse: box('rgba(139,92,246,0.14)', 'rgba(139,92,246,0.28)', 'rgba(109,66,215,0.40)'),
    inspect: box('rgba(56,189,248,0.16)', 'rgba(56,189,248,0.32)', 'rgba(2,132,199,0.44)')
  };

  var VIZ_VIEWS = [
    {
      route: '/visual/production-line', title: '产线展示', current: true,
      desc: '车间等距视图：主机、输送带与 AGV 转运实时状态。'
    },
    {
      route: '/visual', title: '可视化中心总览',
      desc: '八个场景的统一入口与整体运行概览。'
    },
    {
      route: '/visual/industrial-screen', title: '工业大屏',
      desc: '总览型大屏：负荷、能耗与质量趋势联动。'
    },
    {
      route: '/visual/storage-station', title: '储能电站',
      desc: '充放电功率、SOC / SOH 与电池温度监视。'
    },
    {
      route: '/visual/data-center', title: '数据中心仿真',
      desc: '机柜阵列、冷通道与制冷回路仿真。'
    },
    {
      route: '/visual/power-distribution', title: '输配电',
      desc: '杆塔线路、进线与馈线回路运行监视。'
    },
    {
      route: '/visual/instruments', title: '仪表监控',
      desc: '弧表盘、趋势曲线与实时告警联动。'
    },
    {
      route: '/visual/port', title: '港口运输',
      desc: '岸桥、堆场与集卡作业仿真。'
    }
  ];

  /* ======================================================================
     8. 首页数据：63 项寄存器点位规范
     ====================================================================== */

  var ACCESS_LABEL = {
    READ_ONLY: 'READ_ONLY',
    READ_WRITE: 'READ_WRITE',
    CRITICAL_WRITE: 'CRITICAL_WRITE'
  };

  var ACCESS_CLASS = {
    READ_ONLY: 'tag--read',
    READ_WRITE: 'tag--write',
    CRITICAL_WRITE: 'tag--critical'
  };

  function P(reg, type, name, unit, access, map, legacy) {
    return { reg: reg, type: type, name: name, unit: unit, access: access, map: map, legacy: !!legacy };
  }

  var POINTS = [
    // —— 主轴与进给 ——
    P(40001, 'UINT16', '主电机实时旋转速度 (Motor_RPM)', 'r/min', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40002, 'FLOAT32', '主轴轴承温度 (Spindle_Temp)', '°C', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40003, 'FLOAT32', '主轴负载率 (Spindle_Load_Ratio)', '%', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40004, 'UINT16', '工作模式切换 (Op_Mode_Select)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40005, 'FLOAT32', '进给速率指令 (Feed_Rate_Cmd)', 'mm/min', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40006, 'FLOAT32', '进给速率反馈 (Feed_Rate_Actual)', 'mm/min', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40007, 'UINT16', '主轴正反转控制 (Spindle_Direction)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40008, 'FLOAT32', '主轴扭矩估算 (Spindle_Torque_Nm)', 'N·m', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40009, 'UINT16', '主轴启停命令 (Spindle_Start_Stop)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40010, 'UINT32', '紧急制动标志位 (Emergency_EStop_Bit)', 'BOOL', 'CRITICAL_WRITE', 'modbus_tcp.write_point'),
    P(40011, 'FLOAT32', '冷却液流量 (Coolant_Flow_Lpm)', 'L/min', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40012, 'FLOAT32', '三轴振动加速度 RMS (Vibration_RMS_G)', 'm/s²', 'READ_ONLY', 'v1.modbus.read', true),
    // —— 轴控与伺服 ——
    P(40013, 'FLOAT32', 'X 轴位置反馈 (Axis_X_Position)', 'mm', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40014, 'FLOAT32', 'Y 轴位置反馈 (Axis_Y_Position)', 'mm', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40015, 'FLOAT32', 'Z 轴位置反馈 (Axis_Z_Position)', 'mm', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40016, 'FLOAT32', 'X 轴跟随误差 (Axis_X_Follow_Err)', 'mm', 'READ_ONLY', 'v1.modbus.read', true),
    P(40017, 'FLOAT32', 'Y 轴跟随误差 (Axis_Y_Follow_Err)', 'mm', 'READ_ONLY', 'v1.modbus.read', true),
    P(40018, 'FLOAT32', 'Z 轴跟随误差 (Axis_Z_Follow_Err)', 'mm', 'READ_ONLY', 'v1.modbus.read', true),
    P(40019, 'UINT16', '伺服使能位掩码 (Servo_Enable_Mask)', '-', 'CRITICAL_WRITE', 'modbus_tcp.write_point'),
    P(40020, 'INT16', '伺服驱动报警代码 (Servo_Alarm_Code)', '-', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40021, 'FLOAT32', 'X 轴速度反馈 (Axis_X_Velocity)', 'mm/s', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40022, 'FLOAT32', 'Y 轴速度反馈 (Axis_Y_Velocity)', 'mm/s', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40023, 'FLOAT32', 'Z 轴速度反馈 (Axis_Z_Velocity)', 'mm/s', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40024, 'UINT16', '回零状态机当前态 (Zero_Return_State)', '-', 'READ_ONLY', 'v1.modbus.read', true),
    // —— 刀具与工艺 ——
    P(40025, 'UINT16', '当前主轴刀具号 (Tool_Number_Active)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40026, 'UINT16', '刀库目标位置 (Tool_Magazine_Target)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40027, 'FLOAT32', '刀具寿命剩余 (Tool_Life_Remain)', '%', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40028, 'UINT16', '刀具磨损超限告警 (Tool_Wear_Alarm)', '-', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40029, 'FLOAT32', '切削进给倍率 (Feed_Override_Ratio)', '%', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40030, 'FLOAT32', '主轴转速倍率 (Spindle_Override_Ratio)', '%', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40031, 'UINT16', '当前执行程序段号 (Nc_Segment_Current)', '-', 'READ_ONLY', 'v1.modbus.read', true),
    P(40032, 'UINT16', '累计加工完成件数 (Production_Count_Pcs)', 'pcs', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40033, 'FLOAT32', '单件加工节拍耗时 (Cycle_Time_Sec)', 's', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40034, 'UINT16', '报警复位命令 (Alarm_Reset_Cmd)', '-', 'CRITICAL_WRITE', 'modbus_tcp.write_point'),
    P(40035, 'UINT32', '累计运行时间 (Runtime_Accum_Sec)', 's', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40036, 'FLOAT32', '气源供给压力 (Air_Pressure_Bar)', 'bar', 'READ_ONLY', 'v1.modbus.read', true),
    // —— 液压 / 气动 / 辅助 ——
    P(40037, 'FLOAT32', '液压油温 (Hydraulic_Oil_Temp)', '°C', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40038, 'FLOAT32', '液压系统压力 (Hydraulic_Pressure_Bar)', 'bar', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40039, 'UINT16', '液压泵启停控制 (Hydraulic_Pump_Cmd)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40040, 'FLOAT32', '集中润滑液位 (Lubricant_Level_Pct)', '%', 'READ_ONLY', 'v1.modbus.read', true),
    P(40041, 'UINT16', '集中润滑周期设定 (Lubricant_Cycle_Set)', 'min', 'READ_WRITE', 'v1.modbus.write', true),
    P(40042, 'FLOAT32', '气动夹具夹紧力 (Clamp_Force_N)', 'N', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40043, 'UINT16', '夹具夹紧 / 松开命令 (Clamp_Open_Close)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40044, 'FLOAT32', '排屑器电机电流 (Chip_Conveyor_Current_A)', 'A', 'READ_ONLY', 'v1.modbus.read', true),
    P(40045, 'BOOL', '排屑器堵塞标志位 (Chip_Jam_Flag)', '-', 'READ_ONLY', 'v1.modbus.read', true),
    P(40046, 'FLOAT32', '电柜内部温度 (Cabinet_Temp)', '°C', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40047, 'UINT16', '电柜散热风扇调速 (Cabinet_Fan_Duty)', '%', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40048, 'FLOAT32', '现场环境相对湿度 (Ambient_Humidity_RH)', '%RH', 'READ_ONLY', 'v1.modbus.read', true),
    // —— 能耗与电力 ——
    P(40049, 'FLOAT32', '设备总有功功率 (Active_Power_Total_kW)', 'kW', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40050, 'FLOAT32', 'A 相工作电流 (Current_Phase_A_A)', 'A', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40051, 'FLOAT32', 'B 相工作电流 (Current_Phase_B_A)', 'A', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40052, 'FLOAT32', 'C 相工作电流 (Current_Phase_C_A)', 'A', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40053, 'FLOAT32', '直流母线电压 (DC_Bus_Voltage_V)', 'V', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40054, 'FLOAT32', '功率因数 PF (Power_Factor_PF)', '-', 'READ_ONLY', 'v1.modbus.read', true),
    P(40055, 'UINT32', '电能累计读数 (Energy_Meter_Total_kWh)', 'kWh', 'READ_ONLY', 'v1.modbus.read', true),
    P(40056, 'FLOAT32', '伺服系统能耗占比 (Servo_Energy_Ratio_Pct)', '%', 'READ_ONLY', 'v1.modbus.read', true),
    // —— 质量与追溯 ——
    P(40057, 'FLOAT32', '在线测头直径偏差 (Probe_Diameter_Dev_mm)', 'mm', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40058, 'FLOAT32', '表面粗糙度估计 (Surface_Roughness_Ra)', 'μm', 'READ_ONLY', 'v1.modbus.read', true),
    P(40059, 'UINT16', '在线检测合格标志 (Quality_Pass_Flag)', '-', 'READ_ONLY', 'modbus_tcp.read_point'),
    P(40060, 'UINT16', '工件批次号低字 (Batch_Id_Low_Word)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40061, 'UINT16', '工件批次号高字 (Batch_Id_High_Word)', '-', 'READ_WRITE', 'modbus_tcp.write_point'),
    P(40062, 'FLOAT32', 'SPC 过程能力指数 Cpk (SPC_Cpk)', '-', 'READ_ONLY', 'v1.modbus.read', true),
    P(40063, 'UINT16', '追溯数据落盘触发 (Trace_Flush_Trigger)', '-', 'CRITICAL_WRITE', 'v1.modbus.write', true)
  ];

  /* ======================================================================
     9. 首页渲染
     ====================================================================== */

  var activeSubsystem = 0;
  var activeStep = 0;
  var busProtocol = 'mqtt';
  var pipelineTimer = null;

  var DOC_LINKS = [
    {
      code: 'ARCH_GUIDE',
      tone: '',
      title: '后端架构实现指南',
      desc: 'EAN 2.0 节点调度、心跳状态机与集群容灾的后端核心设计。',
      href: './EdgeOS 后端实现指南.html'
    },
    {
      code: 'WIRE_SPEC',
      tone: 'amber',
      title: '通信协议规范 (MQTT/NATS)',
      desc: '基于统一寻址模型的对称通道封装与丢包重放控制。',
      href: './edgeos/EdgeCore通信协议规范(MQTT-NATS).html'
    },
    {
      code: 'AI_PLANNING',
      tone: 'emerald',
      title: 'AI 协同组件规划',
      desc: '协议逆向、文档解析与人工确认回环的组件级规划。',
      href: './edgeos/AI协同组件规划.html'
    },
    {
      code: 'REPO_MAIN',
      tone: 'slate',
      title: 'GitHub 官方源码仓库',
      desc: '获取 EdgeOS 发行包、构建流水线与社区补丁。',
      href: 'https://github.com/anviod/EdgeOS'
    },
    {
      code: 'EAN_GUIDE',
      tone: '',
      title: 'EAN 2.0 改造指南',
      desc: 'edgeCore ↔ EdgeOS 双端改造清单与联调步骤。',
      href: './edgeos/EAN2.0-EdgeCore-EdgeOS改造指南.html'
    },
    {
      code: 'EDGECORE_PORTAL',
      tone: 'emerald',
      title: 'edgeCore 节点运行时',
      desc: '面向嵌入式与工业微机的端侧原生轻量运行时文档。',
      href: 'https://anviod.github.io/edgeCore/'
    },
    {
      code: 'TEST_PLAN',
      tone: 'amber',
      title: '通信测试方案',
      desc: 'EdgeCore 与 EdgeOS 端到端联调与验收用例。',
      href: './edgeos/EdgeCore与EdgeOS通信测试方案.html'
    },
    {
      code: 'STYLE_GUIDE',
      tone: 'slate',
      title: '样式规范',
      desc: '文档站点的排版、配色与组件使用约定。',
      href: './样式规范.html'
    }
  ];

  function renderHeroDocs() {
    var grid = $('#doc-grid');
    if (!grid) return;
    grid.innerHTML = DOC_LINKS.map(function (d) {
      return '' +
        '<a class="doc-card glass spring" href="' + escapeAttr(d.href) + '">' +
        '<div class="doc-card__code' + (d.tone ? ' doc-card__code--' + d.tone : '') + '">' + escapeHtml(d.code) + '</div>' +
        '<h3>' + escapeHtml(d.title) + '</h3>' +
        '<p>' + escapeHtml(d.desc) + '</p>' +
        '</a>';
    }).join('');
  }

  function renderScenarios() {
    var grid = $('#scenario-grid');
    if (!grid) return;

    grid.innerHTML = SCENARIOS.map(function (s) {
      return '' +
        '<article class="scenario-card glass">' +
        '<div class="scenario-card__head">' +
        '<span class="scenario-card__no">' + escapeHtml(s.no) + '</span>' +
        '<h3>' + escapeHtml(s.title) + '</h3>' +
        '</div>' +
        '<p class="scenario-card__desc">' + escapeHtml(s.desc) + '</p>' +
        '<ul class="scenario-card__points">' +
        s.points.map(function (p) {
          return '<li>' + escapeHtml(p) + '</li>';
        }).join('') +
        '</ul>' +
        '<div class="scenario-card__caps">' +
        s.caps.map(function (c) {
          return '<span class="scenario-card__cap">' + escapeHtml(c) + '</span>';
        }).join('') +
        '</div>' +
        '</article>';
    }).join('');
  }

  /* —— 2.5D 等轴场景 —— */

  var isoHudStore = [];

  function registerHud(data) {
    isoHudStore.push(isoHudHtml(data));
    return isoHudStore.length - 1;
  }

  function isoHudHtml(d) {
    return '' +
      '<div class="iso-hud__top">' +
      '<span class="iso-hud__title">' + escapeHtml(d.title) + '</span>' +
      (d.badge
        ? '<span class="iso-hud__badge" style="color:' + d.accent +
          ';background:' + d.accent + '1f">' + escapeHtml(d.badge) + '</span>'
        : '') +
      '</div>' +
      d.rows.map(function (r) {
        return '<div class="iso-hud__row">' +
          '<span>' + escapeHtml(r.label) + '</span>' +
          '<b>' + escapeHtml(r.value) + '</b>' +
          '</div>';
      }).join('') +
      (d.foot ? '<div class="iso-hud__foot">' + escapeHtml(d.foot) + '</div>' : '');
  }

  /**
   * 一个等轴长方体：平铺面（width × depth）+ 抬升 height。
   * 三片分别落在顶面 translateZ(h)、前面 rotateX(90deg)、侧面 rotateY(-90deg)。
   */
  function isoCube(x, y, w, d, h, style, opts) {
    var o = opts || {};
    var hudIdx = o.hud ? ' data-hud="' + registerHud(o.hud) + '"' : '';
    return '' +
      '<div class="iso-cube" style="left:' + x + 'px;top:' + y + 'px;width:' + w +
      'px;height:' + d + 'px;--h:' + h + 'px"' + hudIdx + '>' +
      '<span class="iso-face iso-face--roof" style="background:' + style.top + '"></span>' +
      '<span class="iso-face iso-face--front" style="background:' + style.front + ';' +
      (o.glow ? 'box-shadow:0 0 26px ' + style.front : '') + '"></span>' +
      '<span class="iso-face iso-face--side" style="background:' + style.side + ';' +
      (o.glow ? 'box-shadow:0 0 18px ' + style.side : '') + '"></span>' +
      (o.beacon
        ? '<span class="iso-cube__beacon" style="background:' + o.beaconColor +
          ';box-shadow:0 0 10px ' + o.beaconColor + '"></span>'
        : '') +
      (o.label ? '<span class="iso-cube__label">' + escapeHtml(o.label) + '</span>' : '') +
      '</div>';
  }

  var ISO_BELT_PALETTE = ['#38BDF8', '#34D399', '#FBBF24', '#A78BFA', '#F472B6', '#22D3EE'];

  function isoBelt(x, y, w, d, h, color, speed, itemCount) {
    var items = '';
    for (var i = 0; i < itemCount; i++) {
      var c = ISO_BELT_PALETTE[i % ISO_BELT_PALETTE.length];
      items += '' +
        '<span class="iso-belt__item" style="left:' + (w / itemCount) * i + 6 +
        'px;--idly:' + (speed / itemCount) * i + 's">' +
        '<span class="iso-belt__item-roof" style="background:' + c + '"></span>' +
        '<span class="iso-belt__item-front" style="background:' + c + 'cc"></span>' +
        '<span class="iso-belt__item-side" style="background:' + c + '99"></span>' +
        '</span>';
    }
    var hudIdx = registerHud({
      title: '输送带',
      accent: color,
      rows: [
        { label: '长度', value: w + ' px' },
        { label: '节拍', value: speed + ' s' },
        { label: '在线工件', value: itemCount + ' 件' }
      ],
      foot: '输送带运行中'
    });
    return '' +
      '<div class="iso-belt" data-hud="' + hudIdx + '" style="left:' + x + 'px;top:' + y +
      'px;width:' + w + 'px;height:' + d + 'px;--bh:' + h + 'px;--belt:' + (w - 24) +
      'px;--dur:' + speed + 's">' +
      '<span class="iso-face iso-face--roof" style="background:' + color +
      '26;border:1px solid ' + color + '55"></span>' +
      '<span class="iso-face iso-face--front" style="background:' + color + '44"></span>' +
      '<span class="iso-face iso-face--side" style="background:' + color + '33"></span>' +
      items +
      '</div>';
  }

  function isoFlowDot(y, left, range, dur, delay, color) {
    return '' +
      '<span class="iso-dot" style="top:' + y + 'px;left:' + left +
      'px;--dh:8px;--dc:' + color + ';--dr:' + range + 'px;--dd:' + dur +
      's;--dly:' + delay + 's">' +
      '<span class="iso-dot__face iso-dot__face--roof" style="background:' + color +
      '55;border:1px solid ' + color + '"></span>' +
      '<span class="iso-dot__face iso-dot__face--front" style="background:' + color + '99"></span>' +
      '<span class="iso-dot__face iso-dot__face--side" style="background:' + color + 'bb"></span>' +
      '</span>';
  }

  function machineHud(m) {
    var st = ISO_STATUS[m.status];
    return {
      title: m.name + ' · ' + m.id,
      accent: st.color,
      badge: st.label,
      rows: [
        { label: 'OEE', value: m.oee.toFixed(1) + '%' },
        { label: '节拍', value: m.rate + ' 件/分' },
        { label: '温度', value: m.temp.toFixed(1) + '℃' }
      ],
      foot: st.foot
    };
  }

  function renderIsoScene() {
    var world = $('#iso-world');
    if (!world) return;

    var cell = ISO_GRID.cell;
    var w = ISO_GRID.cols * cell;
    var h = ISO_GRID.rows * cell;

    world.style.width = w + 'px';
    world.style.height = h + 'px';
    world.style.marginLeft = (-w / 2) + 'px';
    world.style.marginTop = (-h / 2) + 'px';
    world.style.setProperty('--iso-scale', ISO_GRID.scale);

    isoHudStore = [];
    var out = [
      '<span class="iso-grid-sheet" style="width:' + w + 'px;height:' + h + 'px"></span>',
      '<span class="iso-floor" style="width:' + w + 'px;height:' + h + 'px"></span>'
    ];

    // 车间墙体 + 立体库
    out.push(isoCube(4, 24, 440, 10, 14, ISO_STATIC.wall, {}));
    out.push(isoCube(264, 48, 180, 26, 86, ISO_STATIC.warehouse, { label: '立体库' }));

    // 一号线主机（M1~M3）
    ISO_MACHINES.forEach(function (m) {
      if (m.id === 'M4' || m.id === 'M5') return;
      var st = ISO_STATUS[m.status];
      out.push(isoCube(m.col * cell + 8, m.row * cell, 44, 44, 54, st, {
        glow: st.glow,
        beacon: m.status === 'warn' || m.status === 'fault',
        beaconColor: st.color,
        label: m.id,
        hud: machineHud(m)
      }));
    });

    out.push(isoBelt(0, 168, 336, 18, 9, '#0EA5E9', 6, 5));

    // 二号线主机（M4~M5）
    ISO_MACHINES.forEach(function (m) {
      if (m.id !== 'M4' && m.id !== 'M5') return;
      var st = ISO_STATUS[m.status];
      out.push(isoCube(m.col * cell + 8, m.row * cell, 44, 44, 54, st, {
        glow: st.glow,
        beacon: m.status === 'warn' || m.status === 'fault',
        beaconColor: st.color,
        label: m.id,
        hud: machineHud(m)
      }));
    });

    out.push(isoBelt(264, 216, 120, 18, 9, '#8B5CF6', 5, 3));

    // AGV 转运车
    ISO_AGVS.forEach(function (agv) {
      out.push(isoCube(agv.x - 10, agv.y - 10, 20, 20, 16, {
        top: agv.color + '44', front: agv.color + '88', side: agv.color + 'aa'
      }, {
        glow: true,
        label: agv.id + (agv.load ? ' ●' : ''),
        hud: {
          title: agv.id,
          accent: agv.color,
          badge: agv.load ? '载货' : '空载',
          rows: [
            { label: '当前位置', value: '(' + agv.x + ', ' + agv.y + ')' },
            { label: '目标', value: '(' + agv.tx + ', ' + agv.ty + ')' }
          ],
          foot: 'AGV 自动转运中'
        }
      }));
    });

    // 质检工位
    out.push(isoCube(408, 240, 44, 44, 38, ISO_STATIC.inspect, { label: '质检台' }));

    // AGV 路径流光
    out.push(isoFlowDot(150, 10, 440, 7, 0, '#38BDF8'));
    out.push(isoFlowDot(240, 10, 440, 8.5, 2, '#34D399'));

    world.innerHTML = out.join('');
  }

  function initIsoHud() {
    var stage = $('#iso-stage');
    var hud = $('#iso-hud');
    if (!stage || !hud) return;

    function hide() {
      hud.classList.remove('is-visible');
      hud.removeAttribute('data-for');
    }

    stage.addEventListener('mouseover', function (e) {
      var node = e.target.closest ? e.target.closest('[data-hud]') : null;
      if (!node) return;

      var idx = node.dataset.hud;
      if (hud.dataset.for !== idx) {
        hud.innerHTML = isoHudStore[Number(idx)] || '';
        hud.dataset.for = idx;
      }
      hud.classList.add('is-visible');

      var sr = stage.getBoundingClientRect();
      var nr = node.getBoundingClientRect();
      var hw = hud.offsetWidth;
      var hh = hud.offsetHeight;
      var left = nr.left - sr.left + nr.width / 2 - hw / 2;
      var top = nr.top - sr.top - hh - 10;
      left = Math.max(8, Math.min(left, sr.width - hw - 8));
      if (top < 8) top = nr.bottom - sr.top + 10;
      hud.style.left = left + 'px';
      hud.style.top = top + 'px';
    });

    stage.addEventListener('mouseleave', hide);
  }

  function renderVisualViews() {
    var grid = $('#viz-grid');
    if (!grid) return;

    grid.innerHTML = VIZ_VIEWS.map(function (v) {
      return '' +
        '<article class="viz-card' + (v.current ? ' is-current' : '') + '">' +
        '<span class="viz-card__route">' + escapeHtml(v.route) + '</span>' +
        '<h4>' + escapeHtml(v.title) + '</h4>' +
        '<p>' + escapeHtml(v.desc) + '</p>' +
        (v.current ? '<span class="viz-card__flag">本页已内嵌预览</span>' : '') +
        '</article>';
    }).join('');
  }

  function renderSubsystems() {
    var grid = $('#subsystem-grid');
    if (!grid) return;

    grid.innerHTML = SUBSYSTEMS.map(function (s, i) {
      return '' +
        '<button type="button" class="capability-card glass spring" data-sub="' + i + '" aria-pressed="' + (i === 0) + '">' +
        '<div class="capability-card__head">' +
        '<span class="capability-card__code">' + escapeHtml(s.code) + '</span>' +
        '<span class="capability-card__led"></span>' +
        '</div>' +
        '<div class="capability-card__body">' +
        '<span class="capability-card__glyph" aria-hidden="true">' +
        '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" ' +
        'stroke-linecap="round" stroke-linejoin="round">' + s.glyph + '</svg>' +
        '</span>' +
        '<h3>' + escapeHtml(s.title) + '</h3>' +
        '</div>' +
        '<p>' + escapeHtml(s.brief) + '</p>' +
        '<div class="capability-card__foot">' +
        '<span>' + escapeHtml(s.latency) + '</span>' +
        '<span class="capability-card__go">查看规约 →</span>' +
        '</div>' +
        '</button>';
    }).join('');

    grid.addEventListener('click', function (e) {
      var card = e.target.closest('[data-sub]');
      if (!card) return;
      selectSubsystem(Number(card.dataset.sub));
    });
  }

  function selectSubsystem(idx) {
    var data = SUBSYSTEMS[idx];
    if (!data) return;
    activeSubsystem = idx;

    $$('#subsystem-grid .capability-card').forEach(function (card) {
      var on = Number(card.dataset.sub) === idx;
      card.classList.toggle('is-active', on);
      card.setAttribute('aria-pressed', String(on));
    });

    var code = $('#inspector-code');
    var title = $('#inspector-title');
    var desc = $('#inspector-desc');
    var slo = $('#inspector-slo');
    var pre = $('#inspector-json');

    if (code) code.textContent = data.code;
    if (title) title.textContent = data.title + ' (' + data.sub + ')';
    if (desc) desc.textContent = data.desc;
    if (slo) {
      slo.innerHTML = data.slo.map(function (row) {
        return '<div class="slo__row"><span class="slo__key">' + escapeHtml(row.k) + '</span>' +
          '<span class="slo__val' + (row.amber ? ' slo__val--amber' : '') + '">' + escapeHtml(row.v) + '</span></div>';
      }).join('');
    }
    if (pre) {
      pre.textContent = JSON.stringify(data.payload, null, 2);
      var frame = pre.parentElement;
      if (frame && !prefersReducedMotion()) {
        frame.classList.remove('spring-pop');
        void frame.offsetWidth;
        frame.classList.add('spring-pop');
      }
    }

    setDrawerCollapsed(false);
  }

  function setDrawerCollapsed(collapsed) {
    var drawer = $('#inspector');
    var btn = $('#inspector-toggle');
    var label = $('#inspector-toggle-label');
    if (!drawer) return;
    drawer.classList.toggle('is-collapsed', collapsed);
    if (btn) btn.setAttribute('aria-expanded', String(!collapsed));
    if (label) label.textContent = collapsed ? '展开抽屉' : '收起抽屉';
  }

  function toggleDrawer() {
    var drawer = $('#inspector');
    if (!drawer) return;
    var collapsed = !drawer.classList.contains('is-collapsed');
    setDrawerCollapsed(collapsed);
    showToast(collapsed ? '已收起控制平面抽屉' : '已展开控制平面抽屉', 'info');
  }

  /* —— 双总线模拟器 —— */

  function topicFor(proto) {
    return proto === 'nats'
      ? '$edgeos.invoke.edgeCore-node-001'
      : '$edgeos/invoke/edgeCore-node-001';
  }

  function renderBusPresets() {
    var box = $('#bus-presets');
    if (!box) return;

    box.innerHTML = BUS_PRESETS.map(function (p) {
      return '' +
        '<button type="button" class="preset-chip" data-preset="' + p.key + '">' +
        '<span class="preset-chip__label">' + escapeHtml(p.label) + '</span>' +
        '<span class="preset-chip__hint">' + escapeHtml(p.hint) + '</span>' +
        '</button>';
    }).join('');

    box.addEventListener('click', function (e) {
      var chip = e.target.closest('[data-preset]');
      if (chip) applyBusPreset(chip.dataset.preset);
    });
  }

  function applyBusPreset(key, silent) {
    var preset = null;
    BUS_PRESETS.forEach(function (p) { if (p.key === key) preset = p; });
    if (!preset) return;

    $$('.preset-chip').forEach(function (chip) {
      chip.classList.toggle('is-active', chip.dataset.preset === key);
    });

    var payload = $('#bus-input-payload');
    if (payload) payload.value = JSON.stringify(preset.body, null, 2);

    if (silent) return;
    pushWire(
      '<span class="ts">[' + nowTime() + ']</span> <span class="c-blue">[PRESET]</span> ' +
      '已载入『' + escapeHtml(preset.label) + '』载荷 · ' + escapeHtml(preset.hint)
    );
    showToast('已载入预设载荷：' + preset.label, 'info');
  }

  function setBusProtocol(proto, silent) {
    busProtocol = proto;

    $$('.proto-switch__btn').forEach(function (btn) {
      btn.classList.toggle('is-active', btn.dataset.proto === proto);
    });

    var status = $('#bus-carrier-status');
    var topicEl = $('#bus-input-topic');

    if (status) {
      status.textContent = proto === 'nats'
        ? 'NATS JetStream · 点分 Subject'
        : 'MQTT 3.1.1 · 斜杠 Topic · QoS 1';
      status.className = 'panel-head__status ' + (proto === 'nats' ? 'is-nats' : 'is-mqtt');
    }
    if (topicEl) topicEl.value = topicFor(proto);

    if (silent) return;

    pushWire(
      '<span class="ts">[' + nowTime() + ']</span> <span class="c-emerald">[CARRIER_SWITCH]</span> 已切换至 ' +
      (proto === 'nats' ? 'NATS JetStream' : 'MQTT 3.1.1') +
      '，统一寻址 ' + (proto === 'nats' ? '/ → .' : '. → /') + ' 已同步重写',
      'c-emerald'
    );
    showToast(proto === 'nats' ? '载波已切换至 NATS JetStream' : '载波已切换至 MQTT 3.1.1 (QoS 1)', 'info');
  }

  function pushWire(html) {
    var stream = $('#bus-wire-stream');
    var box = $('#bus-wire');
    if (!stream) return;
    var line = document.createElement('div');
    line.innerHTML = html;
    stream.appendChild(line);
    if (stream.children.length > 60) stream.removeChild(stream.children[0]);
    if (box) box.scrollTop = box.scrollHeight;
  }

  function fireBusMessage() {
    var topicEl = $('#bus-input-topic');
    var payloadEl = $('#bus-input-payload');
    if (!topicEl) return;

    var raw = payloadEl ? payloadEl.value.trim() : '';
    var time = nowTime();
    var action = 'UNKNOWN';

    try {
      var parsed = JSON.parse(raw);
      action = (parsed.body && parsed.body.capability) || parsed.capability || parsed.action || 'UNKNOWN';
    } catch (e) {
      pushWire(
        '<span class="ts">[' + time + ']</span> <span class="c-red">[SCHEMA_REJECT]</span> ' +
        '控制载荷不是合法 JSON，已按 EAN-2.0-STRICT 阻断下发'
      );
      showToast('控制载荷 JSON 校验失败，已阻断下发', 'error');
      return;
    }

    var latency = (98 + Math.random() * 60).toFixed(0);
    var delim = topicEl.value;

    pushWire(
      '<span class="ts">[' + time + ']</span> <span class="c-cobalt">[PUB]</span> ' +
      delim + ' → ' + escapeHtml(action) + ' · QoS 1'
    );
    pushWire(
      '<span class="ts">[' + time + ']</span> <span class="c-emerald">[REPLY]</span> ' +
      '$edgeos' + (busProtocol === 'nats' ? '.' : '/') + 'reply/edgeos-planner-001 · ' +
      'status=completed · latency_ms=' + latency
    );

    var ackEl = $('#bus-stat-ack');
    var transitEl = $('#bus-stat-transit');
    if (ackEl) ackEl.textContent = 'VERIFIED · invoke-001';
    if (transitEl) transitEl.textContent = 'ROUND_TRIP: ' + latency + 'ms';

    showToast('invoke_capability 已下发双总线 (' + action + ')', 'success');
  }

  function injectFailover() {
    var time = nowTime();
    var alt = busProtocol === 'mqtt'
      ? 'MQTT 3.1.1 / $edgeos/invoke/...'
      : 'NATS JetStream / $edgeos.invoke....';

    pushWire('<span class="ts">[' + time + ']</span> <span class="c-amber">[TIMEOUT]</span> 主通道未在 timeout_sec=10 内回 invoke_response，进入 retry 预算');
    pushWire('<span class="ts">[' + time + ']</span> <span class="c-amber">[RETRY]</span> retry 1/2 · priority=normal · 重发至 ' + escapeHtml(alt));
    pushWire('<span class="ts">[' + time + ']</span> <span class="c-amber">[FAILOVER]</span> 主通道仍无应答，切换至 V1.0 兼容通道 $edgeCore/cmd/{node}/write 转交成功');

    var ackEl = $('#bus-stat-ack');
    var transitEl = $('#bus-stat-transit');
    if (ackEl) ackEl.textContent = 'DEGRADED · V1 FALLBACK OK';
    if (transitEl) transitEl.textContent = 'ROUND_TRIP: ~820ms (V1 通道)';

    showToast('触发链路降级：已通过 V1.0 兼容通道转交成功', 'warn');
  }

  /* —— 六工步流水线 —— */

  function renderPipeline() {
    var grid = $('#pipeline-grid');
    if (!grid) return;

    grid.innerHTML = PIPELINE.map(function (s, i) {
      var gate = i === PIPELINE.length - 1 ? ' is-gate' : '';
      var state = i === 0 ? ' is-active' : ' is-pending';
      var label = i === 0 ? '就绪' : (gate ? '终审门禁' : '等待');
      return '' +
        '<button type="button" class="pipeline-step spring' + gate + state + '" data-step="' + i + '">' +
        '<span class="pipeline-step__num">' + escapeHtml(s.badge) + '</span>' +
        '<span class="pipeline-step__label">' + escapeHtml(s.title.split(' ')[0]) + '</span>' +
        '<span class="pipeline-step__status">' + label + '</span>' +
        '</button>';
    }).join('');

    grid.addEventListener('click', function (e) {
      var btn = e.target.closest('[data-step]');
      if (!btn) return;
      stopPipelineAuto();
      setPipelineStep(Number(btn.dataset.step));
    });

    setPipelineStep(0);
  }

  function setPipelineStep(idx) {
    var step = PIPELINE[idx];
    if (!step) return;
    activeStep = idx;

    $$('#pipeline-grid .pipeline-step').forEach(function (btn) {
      var i = Number(btn.dataset.step);
      var status = btn.querySelector('.pipeline-step__status');
      var gate = i === PIPELINE.length - 1;

      btn.classList.remove('is-active', 'is-done', 'is-pending');

      if (i === idx) {
        btn.classList.add('is-active');
        if (status) status.textContent = '执行中';
      } else if (i < idx) {
        btn.classList.add('is-done');
        if (status) status.textContent = '已通过';
      } else {
        btn.classList.add('is-pending');
        if (status) status.textContent = gate ? '终审门禁' : '等待';
      }
    });

    var badge = $('#pipe-badge');
    var title = $('#pipe-title');
    var text = $('#pipe-text');
    var code = $('#pipe-code');

    if (badge) badge.textContent = step.badge;
    if (title) title.textContent = step.title;
    if (text) text.textContent = step.text;
    if (code) code.textContent = step.code;
  }

  function stopPipelineAuto() {
    if (pipelineTimer) {
      clearInterval(pipelineTimer);
      pipelineTimer = null;
    }
    var btn = $('#btn-run-all');
    if (btn) {
      btn.disabled = false;
      btn.textContent = '一键全自动连贯执行 (AUTO RUN)';
    }
  }

  function runPipelineAll() {
    if (pipelineTimer) { stopPipelineAuto(); return; }

    var btn = $('#btn-run-all');
    if (btn) {
      btn.disabled = true;
      btn.textContent = '执行中… 点击中止';
    }

    var step = 0;
    setPipelineStep(step);

    pipelineTimer = setInterval(function () {
      step++;
      if (step < PIPELINE.length) {
        setPipelineStep(step);
      } else {
        stopPipelineAuto();
        showToast('全流水线六工步已连贯执行验证完毕', 'success');
      }
    }, 900);
  }

  function advanceStep() {
    stopPipelineAuto();
    setPipelineStep((activeStep + 1) % PIPELINE.length);
  }

  function resetPipeline() {
    stopPipelineAuto();
    setPipelineStep(0);
    showToast('工步状态机已重置', 'info');
  }

  /* —— 点位规范库 —— */

  function renderPoints() {
    var tbody = $('#point-table-body');
    if (!tbody) return;

    tbody.innerHTML = POINTS.map(function (p) {
      var accessClass = ACCESS_CLASS[p.access] || 'tag--read';
      var mapCell = p.legacy
        ? '<span class="map-legacy">⟲ ' + escapeHtml(p.map) + '</span>'
        : '<span class="map-native">✓ ' + escapeHtml(p.map) + '</span>';

      return '' +
        '<tr data-reg="' + p.reg + '">' +
        '<td class="col-reg">' + p.reg + '</td>' +
        '<td>' + escapeHtml(p.type) + '</td>' +
        '<td class="col-name">' + escapeHtml(p.name) + '</td>' +
        '<td>' + escapeHtml(p.unit) + '</td>' +
        '<td><span class="tag ' + accessClass + '">' + escapeHtml(ACCESS_LABEL[p.access] || p.access) + '</span></td>' +
        '<td class="col-right">' + mapCell + '</td>' +
        '</tr>';
    }).join('');

    updatePointCount(POINTS.length);
  }

  function updatePointCount(visible) {
    var el = $('#point-count');
    if (!el) return;
    var total = POINTS.length;
    el.textContent = visible === total ? '总点位: ' + total : '命中: ' + visible + ' / ' + total;
    el.classList.toggle('is-filtered', visible !== total);
  }

  function filterPoints() {
    var input = $('#point-filter');
    var tbody = $('#point-table-body');
    if (!tbody) return;

    var q = (input ? input.value : '').trim().toLowerCase();
    var rows = $$('#point-table-body tr');
    var visible = 0;

    rows.forEach(function (row) {
      if (row.classList.contains('spec-empty')) return;
      var hit = !q || row.textContent.toLowerCase().indexOf(q) >= 0;
      row.style.display = hit ? '' : 'none';
      if (hit) visible++;
    });

    var empty = $('#point-empty');
    if (visible === 0) {
      if (!empty) {
        empty = document.createElement('tr');
        empty.id = 'point-empty';
        empty.className = 'spec-empty';
        empty.innerHTML = '<td colspan="6">未匹配到寄存器点位 —— 试试 "40001"、"温度"、"read_point" 或 "CRITICAL_WRITE"</td>';
        tbody.appendChild(empty);
      }
      empty.style.display = '';
    } else if (empty) {
      empty.style.display = 'none';
    }

    updatePointCount(visible);
  }

  function focusPointRow(reg) {
    var input = $('#point-filter');
    if (input) input.value = String(reg);
    filterPoints();
    var row = $('#point-table-body tr[data-reg="' + reg + '"]');
    if (row && !prefersReducedMotion()) {
      row.classList.remove('is-flash');
      void row.offsetWidth;
      row.classList.add('is-flash');
    }
  }

  /* ======================================================================
     10. 全局命令面板
     ====================================================================== */

  var paletteAll = [];
  var paletteShown = [];
  var paletteActive = -1;

  function buildPalette() {
    paletteAll = [];

    [
      ['explorer', '架构探查器', '六大控制平面 + 真实 EAN 2.0 报文检视'],
      ['bus', '双总线模拟器', 'MQTT 3.1.1 / NATS JetStream 对称寻址与降级演练'],
      ['pipeline', '流水线工步', '六工步单步步进与一键连贯执行'],
      ['specs', '点位规范库', '63 项寄存器点位与 Capability 映射']
    ].forEach(function (t) {
      paletteAll.push({
        group: '导航',
        title: t[1],
        meta: t[2],
        run: function () { switchTab(t[0]); }
      });
    });

    SUBSYSTEMS.forEach(function (s, i) {
      paletteAll.push({
        group: '控制平面',
        title: s.code.split('/')[0].trim() + ' ' + s.title,
        meta: s.sub + ' · ' + s.latency,
        run: function () { switchTab('explorer'); selectSubsystem(i); }
      });
    });

    PIPELINE.forEach(function (s, i) {
      paletteAll.push({
        group: '工步',
        title: s.badge + ' ' + s.title,
        meta: 'Pipeline · 单步验证',
        run: function () { switchTab('pipeline'); stopPipelineAuto(); setPipelineStep(i); }
      });
    });

    SCENARIOS.forEach(function (s) {
      paletteAll.push({
        group: '场景用例',
        title: s.no + ' ' + s.title,
        meta: s.caps.join(' · '),
        run: function () { switchTab('scenarios'); }
      });
    });

    VIZ_VIEWS.forEach(function (v) {
      paletteAll.push({
        group: '2.5D 可视化',
        title: v.title,
        meta: v.route + ' · ' + v.desc,
        run: function () { switchTab('visualization'); }
      });
    });

    BUS_PRESETS.forEach(function (p) {
      paletteAll.push({
        group: '总线动作',
        title: '载入预设载荷 · ' + p.label,
        meta: 'Bus · ' + p.hint,
        run: function () { switchTab('bus'); applyBusPreset(p.key); }
      });
    });

    [
      { t: '切换到 NATS JetStream 载波', m: 'Bus · 点分 Subject', f: function () { switchTab('bus'); setBusProtocol('nats'); } },
      { t: '切换到 MQTT 3.1.1 载波', m: 'Bus · 斜杠 Topic · QoS 1', f: function () { switchTab('bus'); setBusProtocol('mqtt'); } },
      { t: '下发 invoke_capability 指令', m: 'Bus · modbus_tcp.write_point', f: function () { switchTab('bus'); fireBusMessage(); } },
      { t: '模拟链路超时降级演练', m: 'Bus · V1.0 兼容通道转交', f: function () { switchTab('bus'); injectFailover(); } }
    ].forEach(function (x) {
      paletteAll.push({ group: '总线动作', title: x.t, meta: x.m, run: x.f });
    });

    POINTS.forEach(function (p) {
      paletteAll.push({
        group: '寄存器点位',
        title: p.reg + ' · ' + p.name,
        meta: p.type + ' · ' + p.unit + ' · ' + p.access + ' → ' + p.map,
        run: function () { switchTab('specs'); focusPointRow(p.reg); }
      });
    });

    DOC_LINKS.forEach(function (d) {
      paletteAll.push({
        group: '文档',
        title: d.title,
        meta: d.code + ' · ' + d.desc,
        run: function () { window.open(d.href, '_blank', 'noopener'); }
      });
    });

    paletteAll.push({
      group: '系统',
      title: '切换亮色 / 深色冷轧钛合金模式',
      meta: 'Theme · 偏好写入 localStorage',
      run: function () { var b = $('[data-theme-toggle]'); if (b) b.click(); }
    });
    paletteAll.push({
      group: '系统',
      title: '复制当前控制平面 Payload',
      meta: 'Clipboard · EAN 2.0 JSON',
      run: function () { copyInspectorJson(); }
    });
  }

  function openPalette(show) {
    var el = $('#palette');
    if (!el) return;
    if (show) {
      el.classList.add('is-open');
      var input = $('#palette-input');
      if (input) {
        input.value = '';
        input.focus();
      }
      renderPalette('');
    } else {
      el.classList.remove('is-open');
    }
  }

  function renderPalette(query) {
    var box = $('#palette-results');
    var countEl = $('#palette-count');
    if (!box) return;

    var q = (query || '').trim().toLowerCase();
    paletteShown = (q
      ? paletteAll.filter(function (it) {
          return (it.group + ' ' + it.title + ' ' + it.meta).toLowerCase().indexOf(q) >= 0;
        })
      : paletteAll
    ).slice(0, 60);
    paletteActive = paletteShown.length ? 0 : -1;
    if (box.scrollTop) box.scrollTop = 0;

    if (!paletteShown.length) {
      box.innerHTML = '<div class="palette__empty">未匹配到条目 —— 试试 "40001"、"NATS"、"治理" 或 "温度"</div>';
    } else {
      var html = '';
      var lastGroup = null;
      paletteShown.forEach(function (it, i) {
        if (it.group !== lastGroup) {
          html += '<div class="palette__group">' + escapeHtml(it.group) + '</div>';
          lastGroup = it.group;
        }
        html += '' +
          '<div class="palette__item' + (i === 0 ? ' is-active' : '') + '" data-idx="' + i + '" role="option">' +
          '<div style="min-width:0">' +
          '<div class="palette__item-title">' + escapeHtml(it.title) + '</div>' +
          '<div class="palette__item-meta">' + escapeHtml(it.meta) + '</div>' +
          '</div>' +
          '<span class="palette__go">执行 ↵</span>' +
          '</div>';
      });
      box.innerHTML = html;
    }

    if (countEl) countEl.textContent = paletteShown.length + ' / ' + paletteAll.length + ' 条';
  }

  function setPaletteActive(idx) {
    paletteActive = idx;
    $$('#palette-results .palette__item').forEach(function (el) {
      el.classList.toggle('is-active', Number(el.dataset.idx) === idx);
    });
  }

  function movePalette(delta) {
    if (!paletteShown.length) return;
    paletteActive = (paletteActive + delta + paletteShown.length) % paletteShown.length;
    setPaletteActive(paletteActive);
    var el = $('#palette-results .palette__item.is-active');
    if (el && el.scrollIntoView) el.scrollIntoView({ block: 'nearest' });
  }

  function executePaletteItem(idx) {
    var item = paletteShown[idx];
    if (!item) return;
    openPalette(false);
    setTimeout(function () { item.run(); }, 60);
  }

  function initPalette() {
    var box = $('#palette-results');
    if (!box) return;

    box.addEventListener('click', function (e) {
      var item = e.target.closest('.palette__item');
      if (!item) return;
      executePaletteItem(Number(item.dataset.idx));
    });

    box.addEventListener('mouseover', function (e) {
      var item = e.target.closest('.palette__item');
      if (!item) return;
      setPaletteActive(Number(item.dataset.idx));
    });

    var modal = $('#palette');
    if (modal) {
      modal.addEventListener('click', function (e) {
        if (e.target === modal) openPalette(false);
      });
    }

    var trigger = $('#palette-trigger');
    if (trigger) trigger.addEventListener('click', function () { openPalette(true); });

    // 实时过滤：输入即检索（IME 合成期不打断，避免中文输入被截断）
    var input = $('#palette-input');
    if (input) {
      input.addEventListener('input', function () {
        if (input.isComposing) return;
        renderPalette(input.value);
      });
      input.addEventListener('compositionend', function () {
        renderPalette(input.value);
      });
      input.addEventListener('keydown', function (e) {
        // 回车/上下/ESC 冒泡到 document 统一处理，这里只阻止表单默认提交行为
        if (e.key === 'Enter') e.preventDefault();
      });
    }

    document.addEventListener('keydown', function (e) {
      var open = modal && modal.classList.contains('is-open');

      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        openPalette(!open);
        return;
      }

      if (!open) {
        var tag = (document.activeElement && document.activeElement.tagName) || '';
        if (e.key === '/' && tag !== 'INPUT' && tag !== 'TEXTAREA') {
          e.preventDefault();
          openPalette(true);
        }
        return;
      }

      if (e.key === 'Escape') { e.preventDefault(); openPalette(false); }
      else if (e.key === 'ArrowDown') { e.preventDefault(); movePalette(1); }
      else if (e.key === 'ArrowUp') { e.preventDefault(); movePalette(-1); }
      else if (e.key === 'Enter' && paletteActive >= 0) { e.preventDefault(); executePaletteItem(paletteActive); }
    });
  }

  function copyInspectorJson() {
    var pre = $('#inspector-json');
    if (!pre) return;
    copyText(pre.textContent);
    showToast('EAN 2.0 Payload 已复制到剪贴板', 'success');
  }

  /* ======================================================================
     11. 亚毫秒遥测时钟
     ====================================================================== */

  function initTelemetryClock() {
    var clockEl = $('#telemetry-clock');
    var upEl = $('#telemetry-uptime');
    var jitterEl = $('#telemetry-jitter');
    if (!clockEl) return;

    var baseMs = Date.now();
    var basePerf = performance.now();
    var bootPerf = basePerf;
    var lastTick = basePerf;
    var acc = 0;
    var n = 0;
    var rafId = null;

    function tick() {
      var p = performance.now();
      var ms = baseMs + (p - basePerf);
      var d = new Date(ms);

      var hh = String(d.getHours()).padStart(2, '0');
      var mm = String(d.getMinutes()).padStart(2, '0');
      var ss = String(d.getSeconds()).padStart(2, '0');
      var milli = String(Math.floor(ms % 1000)).padStart(3, '0');
      // 亚毫秒部分：毫秒小数 × 1000 → 微秒读数
      var micro = String(Math.floor((ms % 1) * 1000)).padStart(3, '0');

      clockEl.textContent = hh + ':' + mm + ':' + ss + '.' + milli + micro;

      var delta = p - lastTick;
      lastTick = p;
      if (delta > 0 && jitterEl) {
        acc += Math.abs(delta - 16.667) * 1000;
        n++;
        if (n >= 24) {
          jitterEl.textContent = (acc / n).toFixed(2) + 'µs';
          acc = 0;
          n = 0;
        }
      }

      if (upEl) {
        var up = Math.floor((p - bootPerf) / 1000);
        upEl.textContent = 'T+ 000:' +
          String(Math.floor(up / 3600)).padStart(2, '0') + ':' +
          String(Math.floor((up % 3600) / 60)).padStart(2, '0') + ':' +
          String(up % 60).padStart(2, '0');
      }

      rafId = requestAnimationFrame(tick);
    }

    function resync() {
      baseMs = Date.now();
      basePerf = performance.now();
      lastTick = basePerf;
    }

    document.addEventListener('visibilitychange', function () {
      if (document.hidden) {
        if (rafId) { cancelAnimationFrame(rafId); rafId = null; }
      } else {
        resync();
        if (!rafId) rafId = requestAnimationFrame(tick);
      }
    });

    rafId = requestAnimationFrame(tick);
  }

  function initTelemetryStream() {
    var log = $('#telemetry-log');
    if (!log) return;
    var latencyEl = $('#metric-latency');

    var lines = [
      function (t, l) { return '<span class="ts">[' + t + ']</span> <span class="c-emerald">HEARTBEAT</span> $edgeos/heartbeat/edgeCore-node-001 · ' + l + 'ms'; },
      function (t) { return '<span class="ts">[' + t + ']</span> <span class="c-blue">DISCOVERY</span> capability_descriptor 63 条已入 Registry'; },
      function (t) { return '<span class="ts">[' + t + ']</span> <span class="c-amber">EVENT</span> temperature.changed 42.1 → 45.2 (previous_value 已保留)'; },
      function (t) { return '<span class="ts">[' + t + ']</span> <span class="c-blue">SHADOW</span> $edgeos/state/..../delta 增量已合并'; },
      function (t) { return '<span class="ts">[' + t + ']</span> <span class="c-emerald">REPLY</span> invoke_response status=completed (modbus_tcp.read_point)'; }
    ];

    var i = 0;
    setInterval(function () {
      var latency = (0.62 + Math.random() * 0.4).toFixed(2);
      if (latencyEl) latencyEl.textContent = latency;

      var line = document.createElement('div');
      line.innerHTML = lines[i % lines.length](nowTime(), latency);
      i++;
      log.appendChild(line);
      if (log.children.length > 8) log.removeChild(log.children[0]);
      log.scrollTop = log.scrollHeight;
    }, 3600);
  }

  /* ======================================================================
     13. 启动
     ====================================================================== */

  function initHome() {
    renderHeroDocs();
    renderSubsystems();
    renderPipeline();
    renderPoints();
    renderBusPresets();
    renderScenarios();
    renderIsoScene();
    renderVisualViews();

    setDrawerCollapsed(false);
    selectSubsystem(0);
    setBusProtocol('mqtt', true);
    applyBusPreset('write', true);
    buildPalette();
    initPalette();
    initIsoHud();
    initTelemetryClock();
    initTelemetryStream();

    var toggle = $('#inspector-toggle');
    if (toggle) toggle.addEventListener('click', toggleDrawer);

    $$('.proto-switch__btn').forEach(function (btn) {
      btn.addEventListener('click', function () { setBusProtocol(btn.dataset.proto); });
    });

    var dispatch = $('#btn-dispatch');
    if (dispatch) dispatch.addEventListener('click', fireBusMessage);

    var failover = $('#btn-failover');
    if (failover) failover.addEventListener('click', injectFailover);

    var runAll = $('#btn-run-all');
    if (runAll) runAll.addEventListener('click', runPipelineAll);

    var reset = $('#btn-reset');
    if (reset) reset.addEventListener('click', resetPipeline);

    var advance = $('#btn-advance');
    if (advance) advance.addEventListener('click', advanceStep);

    var filter = $('#point-filter');
    if (filter) {
      filter.addEventListener('input', filterPoints);
      filter.addEventListener('keyup', filterPoints);
    }

    var copyBtn = $('#inspector-copy');
    if (copyBtn) copyBtn.addEventListener('click', copyInspectorJson);

    $$('[data-nav-tab]').forEach(function (btn) {
      btn.addEventListener('click', function () { switchTab(btn.dataset.navTab); });
    });

    /* 顶部胶囊中的滚动型按钮（系统总览 / 场景用例 / 文档）此前没有任何绑定，点击无响应 */
    $$('[data-nav-scroll]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var key = btn.dataset.navScroll;
        Object.keys(SCROLL_TABS).forEach(function (k) {
          if (SCROLL_TABS[k] === key) key = k;
        });
        switchTab(key);
      });
    });

    switchTab('explorer', { skipScroll: true });
  }

  function init() {
    initTheme();

    if ($('#portal-home')) {
      initHome();
    } else {
      highlightCode();
      addCopyButtons();
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // 供内联 onclick 使用的公共接口
  window.EdgeOSPortal = {
    switchTab: switchTab,
    selectSubsystem: selectSubsystem,
    toggleDrawer: toggleDrawer,
    setBusProtocol: setBusProtocol,
    fireBusMessage: fireBusMessage,
    injectFailover: injectFailover,
    runPipelineAll: runPipelineAll,
    resetPipeline: resetPipeline,
    advanceStep: advanceStep,
    openPalette: openPalette,
    copyInspectorJson: copyInspectorJson,
    showToast: showToast
  };
})();
