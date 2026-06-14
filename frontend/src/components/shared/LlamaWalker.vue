<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import gsap from 'gsap'

const emit = defineEmits<{ close: [] }>()

const svgRef   = ref<SVGSVGElement | null>(null)
const cloudRef = ref<SVGSVGElement | null>(null)
const bgRef    = ref<SVGSVGElement | null>(null)
const ctx      = ref<gsap.Context | null>(null)

onMounted(() => {
  if (!svgRef.value || !bgRef.value || !cloudRef.value) return

  ctx.value = gsap.context(() => {
    const svg = svgRef.value!
    const q   = (id: string) => svg.querySelector(`#${id}`)!

    const flLeg   = q('fl-leg')
    const frLeg   = q('fr-leg')
    const blLeg   = q('bl-leg')
    const brLeg   = q('br-leg')
    const bodyGrp = q('body-group')
    const headGrp = q('head-group')
    const tail    = q('tail')
    const earL    = q('ear-left')
    const earR    = q('ear-right')
    const eyeGrp  = q('eye-group')

    // Ноги: pivot — бедро (верхняя точка, внутри тела)
    gsap.set(flLeg, { svgOrigin: '157 94' })
    gsap.set(frLeg, { svgOrigin: '171 94' })
    gsap.set(blLeg, { svgOrigin: '127 94' })
    gsap.set(brLeg, { svgOrigin: '142 94' })
    gsap.set(tail,     { svgOrigin: '118 65'  })
    gsap.set(earL,     { svgOrigin: '200 14'  })
    gsap.set(earR,     { svgOrigin: '213 12'  })
    gsap.set(headGrp,  { svgOrigin: '207 37'  })
    gsap.set(bodyGrp,  { svgOrigin: '153 78'  })
    gsap.set(eyeGrp,   { svgOrigin: '203 19'  })

    // Боковая походка: bl → fl → br → fr, задние ноги ведут передние на 1 такт.
    // Каждая нога — независимый gsap.to с keyframes; цикл 4d, замкнут.
    // Easing: отрыв (power2.in) → перенос→посадка (power2.out) → опора (sine.in+out).
    const d = 0.26

    // Каждая нога — отдельный timeline с fromTo → гарантированный старт и чистый loop.
    // Боковая походка bl→fl→br→fr, каждая смещена на 1 такт (d).
    gsap.timeline({ repeat: -1 })
      .fromTo(blLeg, { rotation: -28 }, { rotation:  0,  duration: d, ease: 'sine.in'    })
      .to(blLeg,                         { rotation:  22, duration: d, ease: 'sine.out'   })
      .to(blLeg,                         { rotation:  0,  duration: d, ease: 'power2.in'  })
      .to(blLeg,                         { rotation: -28, duration: d, ease: 'power2.out' })

    gsap.timeline({ repeat: -1 })
      .fromTo(flLeg, { rotation:  0  }, { rotation: -28, duration: d, ease: 'power2.out' })
      .to(flLeg,                         { rotation:  0,  duration: d, ease: 'sine.in'    })
      .to(flLeg,                         { rotation:  22, duration: d, ease: 'sine.out'   })
      .to(flLeg,                         { rotation:  0,  duration: d, ease: 'power2.in'  })

    gsap.timeline({ repeat: -1 })
      .fromTo(brLeg, { rotation: 22  }, { rotation:  0,  duration: d, ease: 'power2.in'  })
      .to(brLeg,                         { rotation: -28, duration: d, ease: 'power2.out' })
      .to(brLeg,                         { rotation:  0,  duration: d, ease: 'sine.in'    })
      .to(brLeg,                         { rotation:  22, duration: d, ease: 'sine.out'   })

    gsap.timeline({ repeat: -1 })
      .fromTo(frLeg, { rotation:  0  }, { rotation:  22, duration: d, ease: 'sine.out'   })
      .to(frLeg,                         { rotation:  0,  duration: d, ease: 'power2.in'  })
      .to(frLeg,                         { rotation: -28, duration: d, ease: 'power2.out' })
      .to(frLeg,                         { rotation:  0,  duration: d, ease: 'sine.in'    })

    const tl = gsap.timeline({ repeat: -1 })

    // ── Тело: лёгкое покачивание 2× за цикл ─────────────────────────────
    tl.to(bodyGrp, { y: -3, rotation:  1.5, duration: d, ease: 'sine.inOut' }, 0)
      .to(bodyGrp, { y:  0, rotation: -1.2, duration: d, ease: 'sine.inOut' }, d)
      .to(bodyGrp, { y: -3, rotation:  1.5, duration: d, ease: 'sine.inOut' }, 2 * d)
      .to(bodyGrp, { y:  0, rotation: -1.2, duration: d, ease: 'sine.inOut' }, 3 * d)

    // ── Голова: кивок ────────────────────────────────────────────────────
      .to(headGrp, { rotation:  2.5, duration: d, ease: 'sine.inOut' }, 0)
      .to(headGrp, { rotation: -2.5, duration: d, ease: 'sine.inOut' }, d)
      .to(headGrp, { rotation:  2.5, duration: d, ease: 'sine.inOut' }, 2 * d)
      .to(headGrp, { rotation: -2.5, duration: d, ease: 'sine.inOut' }, 3 * d)

    // ── Хвост ────────────────────────────────────────────────────────────
      .to(tail, { rotation:  22, duration: 2 * d, ease: 'sine.inOut' }, 0)
      .to(tail, { rotation:  -6, duration: 2 * d, ease: 'sine.inOut' }, 2 * d)

    // Фоновый пейзаж — медленнее облака (параллакс); цикл = горы+лес = 2400px
    gsap.fromTo(bgRef.value!, { x: 0 }, { x: -5200, duration: 244, ease: 'none', repeat: -1 })

    // Облако + ветер летят справа налево — иллюзия движения альпаки
    const stage  = svg.closest('.llama-stage') as HTMLElement
    const stageW = stage?.offsetWidth ?? 800
    gsap.fromTo(cloudRef.value!, { x: stageW }, { x: -130, duration: 13, ease: 'none', repeat: -1 })

    // Ветер: 3 штриха внутри SVG, перед мордой, затухают не доходя до тела
    const w1 = q('wind-1')
    const w2 = q('wind-2')
    const w3 = q('wind-3')
    // Стартуют за правым краем SVG (+100px), затухают у морды (-12px) — на альпаку не заходят
    gsap.fromTo(w1, { x: 100, opacity: 0.45 }, { x: -12, opacity: 0, duration: 2.8, ease: 'none', repeat: -1, repeatDelay: 2.0              })
    gsap.fromTo(w2, { x: 100, opacity: 0.35 }, { x: -12, opacity: 0, duration: 3.5, ease: 'none', repeat: -1, repeatDelay: 1.2, delay: 1.7  })
    gsap.fromTo(w3, { x: 100, opacity: 0.28 }, { x: -12, opacity: 0, duration: 2.4, ease: 'none', repeat: -1, repeatDelay: 1.8, delay: 0.8  })

    function scheduleBlink() {
      gsap.delayedCall(gsap.utils.random(2.5, 7), () => {
        gsap.timeline()
          .to(eyeGrp, { scaleY: 0, duration: 0.07, ease: 'power2.in'  })
          .to(eyeGrp, { scaleY: 1, duration: 0.13, ease: 'power2.out' })
          .call(scheduleBlink)
      })
    }
    scheduleBlink()

    function scheduleEarFlick() {
      gsap.delayedCall(gsap.utils.random(2, 6), () => {
        const isLeft = Math.random() > 0.5
        const ear    = isLeft ? earL : earR
        const dir    = isLeft ? -1 : 1
        gsap.timeline()
          .to(ear, { rotation: dir * 14, duration: 0.1,  ease: 'power2.out'         })
          .to(ear, { rotation: 0,        duration: 0.32, ease: 'elastic.out(1,.45)' })
          .call(scheduleEarFlick)
      })
    }
    scheduleEarFlick()
  }, svgRef.value)
})

onUnmounted(() => {
  ctx.value?.revert()
})
</script>

<template>
  <div class="relative">
  <button
    class="absolute top-2 right-3 z-10 flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-black/40 hover:bg-black/60 text-white text-xs font-medium pointer-events-auto backdrop-blur-sm"
    title="Скрыть анимацию"
    @click="emit('close')"
  >
    <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
      <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
    </svg>
    Скрыть
  </button>
  <div
    class="llama-stage relative overflow-hidden select-none pointer-events-none"
    style="height: 152px"
    aria-hidden="true"
  >
    <!-- Горный пейзаж — медленный параллакс-фон -->
    <svg
      ref="bgRef"
      class="absolute"
      style="top: 0; left: 0"
      viewBox="0 0 10400 152"
      width="10400"
      height="152"
      xmlns="http://www.w3.org/2000/svg"
    >
      <!-- Паттерн 1200px — два экземпляра через <use> для бесшовного лупа -->
      <defs>
        <g id="mtn">
          <!-- Слой 1: дальние горы — плавные кривые безье, 5 пиков -->
          <path d="M 0,152 L 0,82 Q 50,62 100,42 L 152,22 Q 204,44 256,72 Q 306,50 356,26 L 404,16 Q 454,36 504,68 Q 556,46 606,22 L 654,12 Q 704,36 754,72 Q 806,48 856,22 L 906,14 Q 956,38 1006,72 Q 1058,46 1108,22 L 1146,16 Q 1172,38 1193,68 Q 1197,74 1200,82 L 1200,152 Z"
                fill="#9BBDD2"/>

          <!-- Снежные шапки: кубические безье вдоль склонов -->
          <path d="M 140,34 C 144,27 148,23 152,22 C 156,23 160,27 164,34 Z M 392,28 C 396,22 400,17 404,16 C 408,17 412,22 416,28 Z M 642,24 C 646,18 650,13 654,12 C 658,13 662,18 666,24 Z M 894,26 C 898,20 902,15 906,14 C 910,15 914,20 918,26 Z M 1134,28 C 1138,22 1142,17 1146,16 C 1150,17 1154,22 1158,28 Z"
                fill="white"/>

          <!-- Теневые рёбра правых склонов пиков -->
          <path d="M 152,22 Q 210,44 248,66" stroke="#7A9CB5" stroke-width="1.5" fill="none" opacity="0.5"/>
          <path d="M 404,16 Q 460,38 500,62" stroke="#7A9CB5" stroke-width="1.5" fill="none" opacity="0.5"/>
          <path d="M 654,12 Q 710,34 750,58" stroke="#7A9CB5" stroke-width="1.5" fill="none" opacity="0.5"/>
          <path d="M 906,14 Q 962,36 1002,60" stroke="#7A9CB5" stroke-width="1.5" fill="none" opacity="0.5"/>
          <path d="M 1146,16 Q 1175,36 1194,58" stroke="#7A9CB5" stroke-width="1.5" fill="none" opacity="0.5"/>

          <!-- Слой 2: средний ярус — чуть ниже, мягкие холмы между пиками -->
          <path d="M 0,152 L 0,100 Q 80,84 162,62 L 212,48 Q 262,62 312,90 Q 382,74 452,54 L 502,42 Q 552,56 602,86 Q 672,70 742,50 L 792,38 Q 842,54 892,84 Q 962,68 1032,48 L 1082,38 Q 1132,56 1172,88 Q 1188,96 1196,100 L 1200,100 L 1200,152 Z"
                fill="#7AA4BC"/>

          <!-- Лесная линия: хвойные деревья (силуэт) -->
          <path d="M 0,152 L 0,118 L 8,90 L 16,118 L 24,84 L 32,118 L 40,92 L 48,118 L 56,87 L 64,118 L 72,84 L 80,118 L 88,90 L 96,118 L 104,87 L 112,118 L 120,93 L 128,118 L 136,85 L 144,118 L 152,91 L 160,118 L 168,90 L 176,118 L 184,84 L 192,118 L 200,92 L 208,118 L 216,87 L 224,118 L 232,84 L 240,118 L 248,90 L 256,118 L 264,87 L 272,118 L 280,93 L 288,118 L 296,85 L 304,118 L 312,91 L 320,118 L 328,90 L 336,118 L 344,84 L 352,118 L 360,92 L 368,118 L 376,87 L 384,118 L 392,84 L 400,118 L 408,90 L 416,118 L 424,87 L 432,118 L 440,93 L 448,118 L 456,85 L 464,118 L 472,91 L 480,118 L 488,90 L 496,118 L 504,84 L 512,118 L 520,92 L 528,118 L 536,87 L 544,118 L 552,84 L 560,118 L 568,90 L 576,118 L 584,87 L 592,118 L 600,93 L 608,118 L 616,85 L 624,118 L 632,91 L 640,118 L 648,90 L 656,118 L 664,84 L 672,118 L 680,92 L 688,118 L 696,87 L 704,118 L 712,84 L 720,118 L 728,90 L 736,118 L 744,87 L 752,118 L 760,93 L 768,118 L 776,85 L 784,118 L 792,91 L 800,118 L 808,90 L 816,118 L 824,84 L 832,118 L 840,92 L 848,118 L 856,87 L 864,118 L 872,84 L 880,118 L 888,90 L 896,118 L 904,87 L 912,118 L 920,93 L 928,118 L 936,85 L 944,118 L 952,91 L 960,118 L 968,90 L 976,118 L 984,84 L 992,118 L 1000,92 L 1008,118 L 1016,87 L 1024,118 L 1032,84 L 1040,118 L 1048,90 L 1056,118 L 1064,87 L 1072,118 L 1080,93 L 1088,118 L 1096,85 L 1104,118 L 1112,91 L 1120,118 L 1128,90 L 1136,118 L 1144,84 L 1152,118 L 1160,92 L 1168,118 L 1176,87 L 1184,118 L 1192,84 L 1200,118 L 1200,152 Z"
                fill="#2D5C48"/>

          <!-- Слой 3: ближние холмы — плавные волны у подножия -->
          <path d="M 0,152 L 0,112 C 100,104 200,108 300,112 C 400,116 500,104 600,110 C 700,116 800,105 900,112 C 1000,118 1100,106 1200,112 L 1200,152 Z"
                fill="#5E8AA6"/>

          <!-- Земля (каменистая полоса) -->
          <rect x="0" y="118" width="1200" height="34" fill="#7A96A8"/>

          <!-- Камни на земле -->
          <ellipse cx="95"  cy="128" rx="9"  ry="4"   fill="#506B7A"/>
          <ellipse cx="240" cy="133" rx="6"  ry="3"   fill="#506B7A"/>
          <ellipse cx="420" cy="129" rx="8"  ry="3.5" fill="#506B7A"/>
          <ellipse cx="570" cy="134" rx="5"  ry="2.5" fill="#506B7A"/>
          <ellipse cx="730" cy="128" rx="7"  ry="3.5" fill="#506B7A"/>
          <ellipse cx="880" cy="132" rx="6"  ry="3"   fill="#506B7A"/>
          <ellipse cx="1030" cy="128" rx="9" ry="4"   fill="#506B7A"/>
          <ellipse cx="1165" cy="133" rx="6" ry="3"   fill="#506B7A"/>
        </g>

        <g id="forest">
          <!-- Дальние холмы (лесная дымка) -->
          <path d="M 0,152 L 0,86 Q 100,72 200,80 Q 300,68 400,76 Q 500,72 600,82 Q 700,68 800,76 Q 900,72 1000,80 Q 1100,68 1200,78 L 1200,152 Z"
                fill="#6A9070"/>

          <!-- Рёбра дальних холмов -->
          <path d="M 200,80 Q 250,73 300,77" stroke="#4E7056" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 400,76 Q 450,69 500,73" stroke="#4E7056" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 600,82 Q 650,73 700,77" stroke="#4E7056" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 800,76 Q 850,69 900,73" stroke="#4E7056" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 1000,80 Q 1050,73 1100,77" stroke="#4E7056" stroke-width="1.2" fill="none" opacity="0.5"/>

          <!-- Отдельные ели: ствол + 3 яруса (нижний темнее, верхний светлее) -->
          <!-- T1 cx=50  yt=12 --> <rect x="47"   y="98"  width="6" height="12" fill="#2A1505"/><path d="M 24,98 L 50,62 L 76,98 Z" fill="#1A4422"/><path d="M 32,70 L 50,34 L 68,70 Z" fill="#22582E"/><path d="M 40,42 L 50,12 L 60,42 Z" fill="#2E6A3A"/>
          <!-- T2 cx=140 yt=18 --> <rect x="137"  y="104" width="6" height="6"  fill="#2A1505"/><path d="M 114,104 L 140,68 L 166,104 Z" fill="#1A4422"/><path d="M 122,76 L 140,40 L 158,76 Z" fill="#22582E"/><path d="M 130,48 L 140,18 L 150,48 Z" fill="#2E6A3A"/>
          <!-- T3 cx=228 yt=7  --> <rect x="225"  y="93"  width="6" height="17" fill="#2A1505"/><path d="M 202,93 L 228,57 L 254,93 Z" fill="#1A4422"/><path d="M 210,65 L 228,29 L 246,65 Z" fill="#22582E"/><path d="M 218,37 L 228,7 L 238,37 Z" fill="#2E6A3A"/>
          <!-- T4 cx=320 yt=15 --> <rect x="317"  y="101" width="6" height="9"  fill="#2A1505"/><path d="M 294,101 L 320,65 L 346,101 Z" fill="#1A4422"/><path d="M 302,73 L 320,37 L 338,73 Z" fill="#22582E"/><path d="M 310,45 L 320,15 L 330,45 Z" fill="#2E6A3A"/>
          <!-- T5 cx=415 yt=5  --> <rect x="412"  y="91"  width="6" height="19" fill="#2A1505"/><path d="M 389,91 L 415,55 L 441,91 Z" fill="#1A4422"/><path d="M 397,63 L 415,27 L 433,63 Z" fill="#22582E"/><path d="M 405,35 L 415,5 L 425,35 Z" fill="#2E6A3A"/>
          <!-- T6 cx=505 yt=17 --> <rect x="502"  y="103" width="6" height="7"  fill="#2A1505"/><path d="M 479,103 L 505,67 L 531,103 Z" fill="#1A4422"/><path d="M 487,75 L 505,39 L 523,75 Z" fill="#22582E"/><path d="M 495,47 L 505,17 L 515,47 Z" fill="#2E6A3A"/>
          <!-- T7 cx=598 yt=10 --> <rect x="595"  y="96"  width="6" height="14" fill="#2A1505"/><path d="M 572,96 L 598,60 L 624,96 Z" fill="#1A4422"/><path d="M 580,68 L 598,32 L 616,68 Z" fill="#22582E"/><path d="M 588,40 L 598,10 L 608,40 Z" fill="#2E6A3A"/>
          <!-- T8 cx=692 yt=14 --> <rect x="689"  y="100" width="6" height="10" fill="#2A1505"/><path d="M 666,100 L 692,64 L 718,100 Z" fill="#1A4422"/><path d="M 674,72 L 692,36 L 710,72 Z" fill="#22582E"/><path d="M 682,44 L 692,14 L 702,44 Z" fill="#2E6A3A"/>
          <!-- T9 cx=788 yt=6  --> <rect x="785"  y="92"  width="6" height="18" fill="#2A1505"/><path d="M 762,92 L 788,56 L 814,92 Z" fill="#1A4422"/><path d="M 770,64 L 788,28 L 806,64 Z" fill="#22582E"/><path d="M 778,36 L 788,6 L 798,36 Z" fill="#2E6A3A"/>
          <!-- T10 cx=878 yt=18--> <rect x="875"  y="104" width="6" height="6"  fill="#2A1505"/><path d="M 852,104 L 878,68 L 904,104 Z" fill="#1A4422"/><path d="M 860,76 L 878,40 L 896,76 Z" fill="#22582E"/><path d="M 868,48 L 878,18 L 888,48 Z" fill="#2E6A3A"/>
          <!-- T11 cx=970 yt=8  --> <rect x="967"  y="94"  width="6" height="16" fill="#2A1505"/><path d="M 944,94 L 970,58 L 996,94 Z" fill="#1A4422"/><path d="M 952,66 L 970,30 L 988,66 Z" fill="#22582E"/><path d="M 960,38 L 970,8 L 980,38 Z" fill="#2E6A3A"/>
          <!-- T12 cx=1062 yt=13--> <rect x="1059" y="99"  width="6" height="11" fill="#2A1505"/><path d="M 1036,99 L 1062,63 L 1088,99 Z" fill="#1A4422"/><path d="M 1044,71 L 1062,35 L 1080,71 Z" fill="#22582E"/><path d="M 1052,43 L 1062,13 L 1072,43 Z" fill="#2E6A3A"/>
          <!-- T13 cx=1155 yt=16--> <rect x="1152" y="102" width="6" height="8"  fill="#2A1505"/><path d="M 1129,102 L 1155,66 L 1181,102 Z" fill="#1A4422"/><path d="M 1137,74 L 1155,38 L 1173,74 Z" fill="#22582E"/><path d="M 1145,46 L 1155,16 L 1165,46 Z" fill="#2E6A3A"/>

          <!-- Подлесок — кустарники у основания (мягкие Q-дуги) -->
          <path d="M 0,152 L 0,110 Q 40,102 80,108 Q 120,100 160,106 Q 200,102 240,108 Q 280,100 320,104 Q 360,100 400,108 Q 440,102 480,108 Q 520,100 560,105 Q 600,101 640,108 Q 680,102 720,108 Q 760,100 800,106 Q 840,102 880,108 Q 920,100 960,105 Q 1000,101 1040,108 Q 1080,102 1120,108 Q 1160,100 1200,106 L 1200,152 Z"
                fill="#3A6A42"/>

          <!-- Земля (лесная подстилка) -->
          <rect x="0" y="118" width="1200" height="34" fill="#4A6830"/>

          <!-- Упавшие стволы -->
          <rect x="55"   y="120" width="58" height="5" rx="2.5" fill="#3A2510"/>
          <rect x="275"  y="122" width="44" height="4" rx="2"   fill="#3A2510"/>
          <rect x="545"  y="121" width="68" height="5" rx="2.5" fill="#3A2510"/>
          <rect x="790"  y="123" width="50" height="4" rx="2"   fill="#3A2510"/>
          <rect x="1015" y="120" width="62" height="5" rx="2.5" fill="#3A2510"/>

          <!-- Камни и кочки -->
          <ellipse cx="185"  cy="129" rx="7"  ry="3"   fill="#3E5A28"/>
          <ellipse cx="415"  cy="133" rx="5"  ry="2.5" fill="#3E5A28"/>
          <ellipse cx="695"  cy="129" rx="8"  ry="3.5" fill="#3E5A28"/>
          <ellipse cx="935"  cy="132" rx="6"  ry="3"   fill="#3E5A28"/>
          <ellipse cx="1145" cy="130" rx="7"  ry="3"   fill="#3E5A28"/>
        </g>

        <g id="opushka">
          <!-- Дальние холмы (нейтральный серо-зелёный — между горным и лесным) -->
          <path d="M 0,152 L 0,90 Q 100,78 200,86 Q 300,78 400,90 L 400,152 Z"
                fill="#7A9888"/>

          <!-- Рёбра холмов -->
          <path d="M 50,84 Q 100,78 150,82"  stroke="#5E7A68" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 200,86 Q 250,79 300,83"  stroke="#5E7A68" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 310,84 Q 355,79 400,84"  stroke="#5E7A68" stroke-width="1.2" fill="none" opacity="0.5"/>

          <!-- Средний ярус зелени -->
          <path d="M 0,152 L 0,104 Q 100,96 200,102 Q 300,96 400,104 L 400,152 Z"
                fill="#588054"/>

          <!-- Редкие ели (3 дерева, масштаб 0.75 от лесных) -->
          <!-- E1 cx=65  yt=18 --> <rect x="62"  y="83" width="6" height="27" fill="#2A1505"/><path d="M 45,83 L 65,56 L 85,83 Z" fill="#1A4422"/><path d="M 51,62 L 65,35 L 79,62 Z" fill="#22582E"/><path d="M 57,41 L 65,18 L 73,41 Z" fill="#2E6A3A"/>
          <!-- E2 cx=200 yt=12 --> <rect x="197" y="77" width="6" height="33" fill="#2A1505"/><path d="M 180,77 L 200,50 L 220,77 Z" fill="#1A4422"/><path d="M 186,56 L 200,29 L 214,56 Z" fill="#22582E"/><path d="M 192,35 L 200,12 L 208,35 Z" fill="#2E6A3A"/>
          <!-- E3 cx=340 yt=20 --> <rect x="337" y="85" width="6" height="25" fill="#2A1505"/><path d="M 320,85 L 340,58 L 360,85 Z" fill="#1A4422"/><path d="M 326,64 L 340,37 L 354,64 Z" fill="#22582E"/><path d="M 332,43 L 340,20 L 348,43 Z" fill="#2E6A3A"/>

          <!-- Редкий подлесок -->
          <path d="M 0,152 L 0,113 Q 60,107 120,111 Q 180,107 240,111 Q 300,107 360,111 Q 380,109 400,111 L 400,152 Z"
                fill="#496A3E"/>

          <!-- Земля (нейтральный луговой тон) -->
          <rect x="0" y="118" width="400" height="34" fill="#5E7E52"/>

          <!-- Камни -->
          <ellipse cx="42"  cy="128" rx="7"  ry="3.5" fill="#4A6050"/>
          <ellipse cx="290" cy="132" rx="5"  ry="2.5" fill="#4A6050"/>

          <!-- Травяные кочки -->
          <ellipse cx="118" cy="126" rx="6"  ry="2.5" fill="#486238"/>
          <ellipse cx="245" cy="130" rx="5"  ry="2"   fill="#486238"/>
          <ellipse cx="374" cy="127" rx="6"  ry="2.5" fill="#486238"/>
        </g>

        <g id="desert">
          <!-- Дальние дюны (светлое золото) -->
          <path d="M 0,152 L 0,82 Q 150,64 300,76 Q 450,58 600,70 Q 750,62 900,74 Q 1050,58 1200,72 L 1200,152 Z"
                fill="#D4A862"/>

          <!-- Гребни дюн (детали) -->
          <path d="M 100,68 Q 225,62 350,70" stroke="#C49040" stroke-width="1.2" fill="none" opacity="0.45"/>
          <path d="M 350,62 Q 475,56 600,64" stroke="#C49040" stroke-width="1.2" fill="none" opacity="0.45"/>
          <path d="M 600,64 Q 725,58 850,66" stroke="#C49040" stroke-width="1.2" fill="none" opacity="0.45"/>
          <path d="M 850,60 Q 975,54 1100,62" stroke="#C49040" stroke-width="1.2" fill="none" opacity="0.45"/>

          <!-- Средние дюны -->
          <path d="M 0,152 L 0,96 Q 200,82 400,92 Q 600,80 800,90 Q 1000,80 1200,90 L 1200,152 Z"
                fill="#C08040"/>

          <!-- Скальные останцы (угловатые породы) -->
          <path d="M 320,108 L 332,92 L 346,86 L 358,90 L 374,108 Z" fill="#8A5220"/>
          <path d="M 650,108 L 663,93 L 676,86 L 688,92 L 702,108 Z" fill="#8A5220"/>
          <path d="M 920,108 L 934,89 L 949,83 L 961,89 L 975,108 Z" fill="#8A5220"/>

          <!-- Кактусы (4 сагуаро: ствол + 2 руки) -->
          <!-- C1 cx=180 --> <rect x="175" y="36" width="10" height="72" rx="5" fill="#2A5A18"/><rect x="160" y="56" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="160" y="38" width="10" height="26" rx="5" fill="#2A5A18"/><rect x="180" y="60" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="190" y="44" width="10" height="24" rx="5" fill="#2A5A18"/>
          <!-- C2 cx=480 --> <rect x="475" y="40" width="10" height="68" rx="5" fill="#2A5A18"/><rect x="460" y="60" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="460" y="44" width="10" height="24" rx="5" fill="#2A5A18"/><rect x="480" y="64" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="490" y="48" width="10" height="24" rx="5" fill="#2A5A18"/>
          <!-- C3 cx=760 --> <rect x="755" y="34" width="10" height="74" rx="5" fill="#2A5A18"/><rect x="740" y="54" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="740" y="36" width="10" height="26" rx="5" fill="#2A5A18"/><rect x="760" y="58" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="770" y="40" width="10" height="26" rx="5" fill="#2A5A18"/>
          <!-- C4 cx=1020--> <rect x="1015" y="38" width="10" height="70" rx="5" fill="#2A5A18"/><rect x="1000" y="58" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="1000" y="42" width="10" height="24" rx="5" fill="#2A5A18"/><rect x="1020" y="62" width="20" height="8" rx="4" fill="#2A5A18"/><rect x="1030" y="46" width="10" height="24" rx="5" fill="#2A5A18"/>

          <!-- Ближние дюны -->
          <path d="M 0,152 L 0,110 C 150,104 300,108 450,104 C 600,108 750,104 900,108 C 1050,104 1200,108 L 1200,152 Z"
                fill="#A8721A"/>

          <!-- Земля (песок) -->
          <rect x="0" y="118" width="1200" height="34" fill="#C0903A"/>

          <!-- Камни на земле -->
          <ellipse cx="80"   cy="128" rx="10" ry="4.5" fill="#7A4818"/>
          <ellipse cx="265"  cy="132" rx="7"  ry="3"   fill="#7A4818"/>
          <ellipse cx="555"  cy="128" rx="9"  ry="4"   fill="#7A4818"/>
          <ellipse cx="845"  cy="131" rx="6"  ry="3"   fill="#7A4818"/>
          <ellipse cx="1105" cy="129" rx="8"  ry="3.5" fill="#7A4818"/>
        </g>

        <g id="sea">
          <!-- Дальнее море (горизонт) -->
          <path d="M 0,152 L 0,72 Q 100,64 200,70 Q 300,62 400,68 Q 500,64 600,70 Q 700,62 800,68 Q 900,64 1000,70 Q 1100,62 1200,68 L 1200,152 Z"
                fill="#5BA8CC"/>

          <!-- Блики на воде (световые дорожки) -->
          <path d="M 50,68 Q 150,62 250,68"  stroke="#7ACAE0" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 300,64 Q 400,58 500,64"  stroke="#7ACAE0" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 550,66 Q 650,60 750,66"  stroke="#7ACAE0" stroke-width="1.2" fill="none" opacity="0.5"/>
          <path d="M 800,64 Q 900,58 1000,64" stroke="#7ACAE0" stroke-width="1.2" fill="none" opacity="0.5"/>

          <!-- Средние волны -->
          <path d="M 0,152 L 0,90 Q 80,82 160,88 Q 240,80 320,86 Q 400,82 480,88 Q 560,80 640,86 Q 720,82 800,88 Q 880,80 960,86 Q 1040,82 1120,88 Q 1160,84 1200,88 L 1200,152 Z"
                fill="#3A88B0"/>

          <!-- Ближние волны -->
          <path d="M 0,152 L 0,108 Q 60,102 120,108 Q 180,102 240,108 Q 300,102 360,108 Q 420,102 480,108 Q 540,102 600,108 Q 660,102 720,108 Q 780,102 840,108 Q 900,102 960,108 Q 1020,102 1080,108 Q 1140,102 1200,108 L 1200,152 Z"
                fill="#2A7898"/>

          <!-- Пена на гребнях волн -->
          <path d="M 0,108 Q 60,102 120,108 Q 180,102 240,108 Q 300,102 360,108 Q 420,102 480,108 Q 540,102 600,108 Q 660,102 720,108 Q 780,102 840,108 Q 900,102 960,108 Q 1020,102 1080,108 Q 1140,102 1200,108"
                stroke="white" stroke-width="2.5" fill="none" opacity="0.6"/>

          <!-- Чайки (M-силуэт, крупные ближе — мелкие дальше) -->
          <path d="M 62,20 C 71,12 76,16 80,18 C 84,16 89,12 98,20"  stroke="#2A5070" stroke-width="2.5" fill="none" stroke-linecap="round"/>
          <path d="M 188,32 C 194,24 197,28 200,30 C 203,28 206,24 212,32" stroke="#2A5070" stroke-width="2.0" fill="none" stroke-linecap="round"/>
          <path d="M 336,24 C 343,16 347,20 350,22 C 353,20 357,16 364,24" stroke="#2A5070" stroke-width="2.0" fill="none" stroke-linecap="round"/>
          <path d="M 480,16 C 490,8 495,12 500,14 C 505,12 510,8 520,16" stroke="#2A5070" stroke-width="2.5" fill="none" stroke-linecap="round"/>
          <path d="M 641,38 C 645,30 648,34 650,36 C 652,34 655,30 659,38" stroke="#2A5070" stroke-width="1.5" fill="none" stroke-linecap="round"/>
          <path d="M 785,22 C 792,14 796,18 800,20 C 804,18 808,14 815,22" stroke="#2A5070" stroke-width="2.0" fill="none" stroke-linecap="round"/>
          <path d="M 939,30 C 944,22 947,26 950,28 C 953,26 956,22 961,30" stroke="#2A5070" stroke-width="2.0" fill="none" stroke-linecap="round"/>
          <path d="M 1082,18 C 1091,10 1096,14 1100,16 C 1104,14 1109,10 1118,18" stroke="#2A5070" stroke-width="2.5" fill="none" stroke-linecap="round"/>

          <!-- Пляж -->
          <rect x="0" y="118" width="1200" height="34" fill="#D4B870"/>

          <!-- Ракушки и камешки -->
          <ellipse cx="95"   cy="127" rx="8"  ry="3.5" fill="#A09060"/>
          <ellipse cx="280"  cy="131" rx="5"  ry="2.5" fill="#A09060"/>
          <ellipse cx="520"  cy="127" rx="7"  ry="3"   fill="#A09060"/>
          <ellipse cx="760"  cy="130" rx="6"  ry="2.5" fill="#A09060"/>
          <ellipse cx="1000" cy="128" rx="8"  ry="3"   fill="#A09060"/>
        </g>
      </defs>

      <!-- горы + опушка + лес + пустыня + море × 2 — бесшовный цикл 5200px -->
      <use href="#mtn"     x="0"    y="0"/>
      <use href="#opushka" x="1200" y="0"/>
      <use href="#forest"  x="1600" y="0"/>
      <use href="#desert"  x="2800" y="0"/>
      <use href="#sea"     x="4000" y="0"/>
      <use href="#mtn"     x="5200" y="0"/>
      <use href="#opushka" x="6400" y="0"/>
      <use href="#forest"  x="6800" y="0"/>
      <use href="#desert"  x="8000" y="0"/>
      <use href="#sea"     x="9200" y="0"/>
    </svg>

    <!--
      ViewBox 240×122. Лама смотрит вправо.
      Тело+шея — ОДИН закрытый путь (нет шва). Все ноги нарисованы
      до тела: тело перекрывает верх ног → плавный переход.
      Голова в профиль с выступающей мордой.
      Заливка #EEF3F5, контур #506878.
    -->
    <!-- Облако летит справа налево -->
    <svg
      ref="cloudRef"
      class="absolute"
      style="top: 2px; left: 0"
      viewBox="0 0 120 48"
      width="120"
      height="48"
      xmlns="http://www.w3.org/2000/svg"
    >
      <!-- Мягкая тень снизу -->
      <ellipse cx="60" cy="43" rx="44" ry="6"  fill="#8AAAC0" opacity="0.18"/>
      <!-- Тело облака -->
      <ellipse cx="60" cy="38" rx="44" ry="11" fill="#BFCFDC"/>
      <circle  cx="36" cy="30" r="15"           fill="#CCD9E5"/>
      <circle  cx="58" cy="24" r="19"           fill="#D8E6F0"/>
      <circle  cx="78" cy="29" r="14"           fill="#CCD9E5"/>
      <ellipse cx="60" cy="37" rx="40" ry="10" fill="#D8E6F0"/>
      <!-- Хайлайт — яркая белая полоса сверху -->
      <ellipse cx="52" cy="21" rx="13" ry="7"  fill="white" opacity="0.55"/>
      <ellipse cx="52" cy="19" rx="8"  ry="4"  fill="white" opacity="0.70"/>
    </svg>

    <svg
      ref="svgRef"
      viewBox="0 0 240 122"
      width="216"
      height="110"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class="absolute"
      style="bottom: 8px; left: 50%; transform: translateX(-50%)"
      overflow="visible"
    >
      <!-- ══════════════════════════════════════════════════════════
           ТЕЛЕЖКА
           ══════════════════════════════════════════════════════════ -->
      <rect x="10" y="66" width="50" height="22" rx="2" fill="#C4904A"/>
      <!-- Тюки (за бортами) -->
      <rect x="13" y="52" width="17" height="34" rx="3" fill="#D4A852"/>
      <line x1="13" y1="63" x2="30" y2="63" stroke="#8A5820" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="13" y1="73" x2="30" y2="73" stroke="#8A5820" stroke-width="1.5" stroke-linecap="round"/>
      <rect x="34" y="54" width="17" height="32" rx="3" fill="#C89040"/>
      <line x1="34" y1="65" x2="51" y2="65" stroke="#8A5820" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="34" y1="75" x2="51" y2="75" stroke="#8A5820" stroke-width="1.5" stroke-linecap="round"/>
      <!-- Мешок (за бортами) -->
      <path d="M 21,74 C 18,66 18,54 20,47 C 22,42 26,39 31,39 C 36,39 40,42 42,47 C 44,54 44,66 41,74 C 38,78 24,78 21,74 Z"
            fill="#C8A870" stroke="#A08040" stroke-width="1"/>
      <path d="M 39,47 C 42,54 42,66 39,74" stroke="#A08040" stroke-width="1.8" fill="none" opacity="0.4"/>
      <path d="M 23,58 C 24,52 23,47 25,44" stroke="#A08040" stroke-width="1" fill="none" opacity="0.35"/>
      <path d="M 23,42 C 25,39 28,37 31,37 C 34,37 37,39 39,42" fill="#B89050"/>
      <path d="M 21,41 C 25,35 37,35 41,41" stroke="#6A4818" stroke-width="2" stroke-linecap="round" fill="none"/>
      <line x1="26" y1="36" x2="24" y2="32" stroke="#6A4818" stroke-width="1.5" stroke-linecap="round"/>
      <line x1="36" y1="36" x2="38" y2="32" stroke="#6A4818" stroke-width="1.5" stroke-linecap="round"/>
      <!-- Борта тележки (поверх груза) -->
      <rect x="10" y="66" width="5"  height="22" fill="#C4904A"/>
      <rect x="55" y="66" width="5"  height="22" fill="#C4904A"/>
      <line x1="21" y1="68" x2="21" y2="86" stroke="#7A5420" stroke-width="1.3"/>
      <line x1="31" y1="68" x2="31" y2="86" stroke="#7A5420" stroke-width="1.3"/>
      <line x1="41" y1="68" x2="41" y2="86" stroke="#7A5420" stroke-width="1.3"/>
      <line x1="51" y1="68" x2="51" y2="86" stroke="#7A5420" stroke-width="1.3"/>
      <rect x="8"  y="62" width="54" height="5" rx="2" fill="#9A6828"/>
      <rect x="8"  y="88" width="54" height="4" rx="2" fill="#9A6828"/>
      <line x1="62" y1="71" x2="152" y2="71" stroke="#9A6828" stroke-width="3"   stroke-linecap="round"/>
      <line x1="62" y1="79" x2="152" y2="79" stroke="#9A6828" stroke-width="3"   stroke-linecap="round"/>
      <rect x="60" y="67"  width="5"  height="14" rx="2" fill="#9A6828"/>
      <!-- Колесо — верхний слой, translate фиксирует центр (0,0) = (34,95) SVG -->
      <g transform="translate(34,95)">
        <g id="wheel">
          <line x1="0"   y1="-18" x2="0"   y2="18"  stroke="#6B4010" stroke-width="1.8"/>
          <line x1="-18" y1="0"   x2="18"  y2="0"   stroke="#6B4010" stroke-width="1.8"/>
          <line x1="-13" y1="-13" x2="13"  y2="13"  stroke="#6B4010" stroke-width="1.8"/>
          <line x1="13"  y1="-13" x2="-13" y2="13"  stroke="#6B4010" stroke-width="1.8"/>
          <circle cx="0" cy="0" r="18" fill="none" stroke="#8A5C22" stroke-width="2.8"/>
          <circle cx="0" cy="0" r="3.5" fill="#3E2408"/>
        </g>
      </g>

      <!-- ══════════════════════════════════════════════════════════
           ЛАМА
           ══════════════════════════════════════════════════════════ -->

      <!-- Хвост: 3 шипа вверх, вершины скруглены Q-дугами -->
      <g id="tail">
        <path d="M 117,69 L 109,59 Q 100,52 110,58 L 109,49 Q 109,41 117,54 L 118,51 Q 119,43 120,57 L 122,66 Z"
              fill="#EEF3F5" stroke="#506878" stroke-width="2.2"
              stroke-linejoin="round"/>
      </g>

      <!-- ──────────────────────────────────────────────────────────
           ВСЕ 4 НОГИ — до тела, тело их перекрывает сверху.
           Каждая нога — path из 2 звеньев (бедро→колено→копыто).
           stroke-linejoin="round" скрывает стык без дополнительных элементов.
           ────────────────────────────────────────────────────────── -->
      <g id="bl-leg">
        <path d="M 127,94 L 125,108 L 127,120" stroke="#506878" stroke-width="9"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <path d="M 127,94 L 125,108 L 127,120" stroke="#EEF3F5" stroke-width="7"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <rect x="122.5" y="117" width="9" height="6" rx="3" fill="#2E3D46"/>
      </g>
      <g id="br-leg">
        <path d="M 142,94 L 140,108 L 142,120" stroke="#506878" stroke-width="9"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <path d="M 142,94 L 140,108 L 142,120" stroke="#EEF3F5" stroke-width="7"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <rect x="137.5" y="117" width="9" height="6" rx="3" fill="#2E3D46"/>
      </g>
      <g id="fl-leg">
        <path d="M 157,94 L 155,108 L 157,120" stroke="#506878" stroke-width="9"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <path d="M 157,94 L 155,108 L 157,120" stroke="#EEF3F5" stroke-width="7"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <rect x="152.5" y="117" width="9" height="6" rx="3" fill="#2E3D46"/>
      </g>
      <g id="fr-leg">
        <path d="M 171,94 L 169,108 L 171,120" stroke="#506878" stroke-width="9"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <path d="M 171,94 L 169,108 L 171,120" stroke="#EEF3F5" stroke-width="7"  stroke-linecap="round" stroke-linejoin="round" fill="none"/>
        <rect x="166.5" y="117" width="9" height="6" rx="3" fill="#2E3D46"/>
      </g>

      <!-- ──────────────────────────────────────────────────────────
           ТЕЛО + ШЕЯ — ОДИН путь (нет шва между шеей и телом).
           Рисуется поверх всех ног → плавный переход.
           Против часовой: рубец → спина (Q-волны) → шея-задняя →
           шея-передняя → грудь → живот (Q-волны) → рубец.
           ────────────────────────────────────────────────────────── -->
      <g id="body-group">
        <path d="M 118,84 C 114,74 115,63 120,59 Q 128,53 136,59 Q 144,53 152,59 Q 160,53 167,58 C 172,56 178,52 184,49 C 190,42 196,37 200,29 C 202,25 206,22 210,23 C 215,25 216,31 214,38 C 210,46 203,55 196,62 C 192,66 188,68 184,70 C 184,79 184,89 180,96 C 178,96 173,99 166,98 Q 155,103 147,97 Q 138,103 129,97 Q 121,103 118,99 C 114,95 113,88 118,82 Z" fill="#EEF3F5" stroke="#506878" stroke-width="2.2" stroke-linejoin="round"/>

        <!-- Упряжь -->
        <path d="M 149,60 C 150,72 149,83 148,96"
              stroke="#7A4418" stroke-width="3.5" stroke-linecap="round" opacity="0.75"/>
        <path d="M 122,78 C 132,80 150,80 158,78"
              stroke="#7A4418" stroke-width="2.5" stroke-linecap="round" opacity="0.65"/>
      </g>

      <!-- ══════════════════════════════════════════════════════════
           ГОЛОВА — профиль с выступающей мордой
           ══════════════════════════════════════════════════════════ -->
      <g id="head-group">

        <!-- Уши (за головой) -->
        <g id="ear-left">
          <path d="M 197,17 C 195,8 200,2 200,0 C 200,2 205,8 203,17 Z"
                fill="#EEF3F5" stroke="#506878" stroke-width="2" stroke-linejoin="miter" stroke-miterlimit="10"/>
          <path d="M 199,16 C 198,9 200,3 200,1"
                stroke="#506878" stroke-width="1" stroke-linecap="round" opacity="0.4"/>
        </g>
        <g id="ear-right">
          <path d="M 211,16 C 209,7 215,2 215,0 C 215,2 221,7 218,16 Z"
                fill="#EEF3F5" stroke="#506878" stroke-width="2" stroke-linejoin="miter" stroke-miterlimit="10"/>
          <path d="M 213,15 C 212,8 215,3 215,1"
                stroke="#506878" stroke-width="1" stroke-linecap="round" opacity="0.4"/>
        </g>

        <!--
          Голова в профиль: единый путь.
          Морда выступает вправо. Подбородок снизу.
          Затылок плавно уходит в шею (шея перекрывает).
        -->
        <path d="M 192,24 C 192,12 200,7 208,7 C 216,7 222,13 226,19 C 229,23 229,28 226,32 C 223,36 218,38 214,37 C 210,38 207,35 206,32 C 203,37 198,37 194,34 C 190,31 190,28 192,24 Z" fill="#EEF3F5" stroke="#506878" stroke-width="2.2" stroke-linejoin="round"/>

        <!-- Закрытый счастливый глаз (профиль) -->
        <g id="eye-group">
          <path d="M 200,20 Q 203,17 206,20"
                stroke="#506878" stroke-width="2.4" stroke-linecap="round" fill="none"/>
          <!-- Ресницы -->
          <line x1="200" y1="20" x2="199" y2="18" stroke="#506878" stroke-width="1.2" stroke-linecap="round"/>
          <line x1="203" y1="18" x2="203" y2="16" stroke="#506878" stroke-width="1.2" stroke-linecap="round"/>
          <line x1="206" y1="19" x2="207" y2="17" stroke="#506878" stroke-width="1.2" stroke-linecap="round"/>
        </g>

        <!-- Солнечные очки — маленькие, плоский верх, чуть скруглённый низ -->
        <g id="sunglasses">
          <!-- Дужка к уху -->
          <path d="M 196,18 C 192,17 190,19 190,22"
                stroke="#16161E" stroke-width="2" stroke-linecap="round" fill="none"/>
          <!-- Линза: плоский верх, чуть скруглённый низ -->
          <path d="M 199,15 L 212,15 C 214,15 214,22 205,22 C 196,22 196,15 199,15 Z"
                fill="#131820" fill-opacity="0.92"/>
          <!-- Полупрозрачный блик-полоска вверху -->
          <path d="M 200,15 L 211,15 L 211,17 Q 205,18 200,17 Z"
                fill="#5090D8" fill-opacity="0.2"/>
          <!-- Оправа -->
          <path d="M 199,15 L 212,15 C 214,15 214,22 205,22 C 196,22 196,15 199,15 Z"
                fill="none" stroke="#16161E" stroke-width="2"/>
          <!-- Блик по верхней кромке -->
          <line x1="200" y1="15.5" x2="211" y2="15.5"
                stroke="white" stroke-width="0.9" stroke-linecap="round" opacity="0.5"/>
        </g>

        <!-- Ноздря -->
        <ellipse cx="226" cy="23" rx="2" ry="1.5" fill="#506878" opacity="0.75"/>

        <!-- Улыбка (под мордой) -->
        <path d="M 218,33 Q 222,37 225,32"
              stroke="#506878" stroke-width="1.6" stroke-linecap="round" fill="none"/>
      </g>

      <!-- Ветер: 3 штриха справа от морды, появляются снаружи viewBox, затухают у морды -->
      <g id="wind-1">
        <path d="M 248,24 C 260,20 274,28 286,24"
              stroke="#6B9DB5" stroke-width="2.5" stroke-linecap="round" fill="none"/>
      </g>
      <g id="wind-2">
        <path d="M 245,16 C 257,12 270,20 282,16"
              stroke="#6B9DB5" stroke-width="2.0" stroke-linecap="round" fill="none"/>
      </g>
      <g id="wind-3">
        <path d="M 250,33 C 261,29 273,37 284,33"
              stroke="#6B9DB5" stroke-width="1.7" stroke-linecap="round" fill="none"/>
      </g>
    </svg>

  </div>
  </div>
</template>

<style scoped>
#wheel {
  transform-box: fill-box;
  transform-origin: center;
  animation: wheel-spin 1.4s linear infinite;
}
@keyframes wheel-spin {
  to { transform: rotate(360deg); }
}
</style>
