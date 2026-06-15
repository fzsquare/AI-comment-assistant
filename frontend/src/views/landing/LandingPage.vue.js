import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { publicApi } from '../../api/public';
const route = useRoute();
const loading = ref(true);
const error = ref('');
const payload = ref(null);
async function load() {
    loading.value = true;
    error.value = '';
    try {
        const { data } = await publicApi.initLanding(String(route.params.token));
        payload.value = data.data;
        if (payload.value) {
            await publicApi.createEvent(String(route.params.token), {
                sessionId: payload.value.sessionId,
                reviewItemId: payload.value.review.id,
                actionType: 'page_view'
            });
        }
    }
    catch (err) {
        error.value = err?.response?.data?.message || '页面加载失败';
    }
    finally {
        loading.value = false;
    }
}
async function copyReview() {
    if (!payload.value)
        return;
    try {
        await navigator.clipboard.writeText(payload.value.review.content);
        await publicApi.createEvent(String(route.params.token), {
            sessionId: payload.value.sessionId,
            reviewItemId: payload.value.review.id,
            actionType: 'review_copy'
        });
        alert('已复制，可直接去平台发布');
    }
    catch {
        alert('复制失败，请手动长按复制');
    }
}
async function switchReview() {
    if (!payload.value)
        return;
    try {
        const { data } = await publicApi.switchReview(String(route.params.token), {
            currentReviewId: payload.value.review.id,
            sessionId: payload.value.sessionId
        });
        payload.value.review = data.data.review;
        payload.value.remainingDispatchableCount = data.data.remainingDispatchableCount;
        await publicApi.createEvent(String(route.params.token), {
            sessionId: payload.value.sessionId,
            reviewItemId: payload.value.review.id,
            actionType: 'review_switch'
        });
    }
    catch (err) {
        alert(err?.response?.data?.message || '暂无推荐文案，请稍后再试');
    }
}
async function jump(link) {
    if (!payload.value)
        return;
    try {
        await navigator.clipboard.writeText(payload.value.review.content);
    }
    catch {
        alert('文案未自动复制，请手动复制后发布');
    }
    await publicApi.createEvent(String(route.params.token), {
        sessionId: payload.value.sessionId,
        reviewItemId: payload.value.review.id,
        actionType: 'platform_link_click',
        platformCode: link.platformCode
    });
    try {
        window.location.href = link.targetUrl;
    }
    catch {
        if (link.backupUrl) {
            window.location.href = link.backupUrl;
            return;
        }
        alert('跳转失败，请稍后重试');
    }
}
onMounted(load);
debugger; /* PartiallyEnd: #3632/scriptSetup.vue */
const __VLS_ctx = {};
let __VLS_components;
let __VLS_directives;
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "page" },
    ...{ style: {} },
});
if (__VLS_ctx.loading) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: "card" },
    });
}
else if (__VLS_ctx.error) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: "card" },
    });
    (__VLS_ctx.error);
}
else if (__VLS_ctx.payload) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: "card" },
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.h1, __VLS_intrinsicElements.h1)({});
    (__VLS_ctx.payload.storeName);
    __VLS_asFunctionalElement(__VLS_intrinsicElements.p, __VLS_intrinsicElements.p)({
        ...{ class: "muted" },
    });
    (__VLS_ctx.payload.primaryPlatformStyle);
    __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ style: {} },
    });
    (__VLS_ctx.payload.review.content);
    __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: "row" },
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
        ...{ onClick: (__VLS_ctx.copyReview) },
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
        ...{ onClick: (__VLS_ctx.switchReview) },
        ...{ class: "secondary" },
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.p, __VLS_intrinsicElements.p)({
        ...{ class: "muted" },
        ...{ style: {} },
    });
    (__VLS_ctx.payload.remainingDispatchableCount);
    if (__VLS_ctx.payload.images.length) {
        __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            ...{ class: "card" },
        });
        __VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
        __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            ...{ class: "row" },
        });
        for (const [image] of __VLS_getVForSourceType((__VLS_ctx.payload.images))) {
            __VLS_asFunctionalElement(__VLS_intrinsicElements.a, __VLS_intrinsicElements.a)({
                key: (image.id),
                href: (image.imageUrl || image.url),
                target: "_blank",
                rel: "noreferrer",
            });
            __VLS_asFunctionalElement(__VLS_intrinsicElements.img)({
                src: (image.thumbnailUrl || image.imageUrl || image.url),
                ...{ style: {} },
            });
        }
    }
    if (__VLS_ctx.payload.platformLinks.length) {
        __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            ...{ class: "card" },
        });
        __VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
        __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            ...{ class: "row" },
        });
        for (const [link] of __VLS_getVForSourceType((__VLS_ctx.payload.platformLinks))) {
            __VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
                ...{ onClick: (...[$event]) => {
                        if (!!(__VLS_ctx.loading))
                            return;
                        if (!!(__VLS_ctx.error))
                            return;
                        if (!(__VLS_ctx.payload))
                            return;
                        if (!(__VLS_ctx.payload.platformLinks.length))
                            return;
                        __VLS_ctx.jump(link);
                    } },
                key: (link.id),
            });
            (link.buttonText);
        }
    }
}
/** @type {__VLS_StyleScopedClasses['page']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['muted']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
/** @type {__VLS_StyleScopedClasses['secondary']} */ ;
/** @type {__VLS_StyleScopedClasses['muted']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
var __VLS_dollars;
const __VLS_self = (await import('vue')).defineComponent({
    setup() {
        return {
            loading: loading,
            error: error,
            payload: payload,
            copyReview: copyReview,
            switchReview: switchReview,
            jump: jump,
        };
    },
});
export default (await import('vue')).defineComponent({
    setup() {
        return {};
    },
});
; /* PartiallyEnd: #4569/main.vue */
