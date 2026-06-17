import { onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { publicApi } from '../../api/public';
const route = useRoute();
const loading = ref(true);
const switching = ref(false);
const error = ref('');
const payload = ref(null);
const selectedTag = ref('');
// 顾客可在发布前编辑成自己的话
const editedContent = ref('');
// 文案变化时，把可编辑框同步成最新文案
watch(() => payload.value?.review.content, (v) => {
    if (v != null)
        editedContent.value = v;
});
async function load() {
    loading.value = true;
    error.value = '';
    try {
        const { data } = await publicApi.initLanding(String(route.params.token));
        payload.value = data.data;
        if (payload.value) {
            editedContent.value = payload.value.review.content;
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
// 顾客点“我点了什么”→ 取一条对应标签的文案
async function pickByTag(keyword) {
    if (!payload.value || switching.value)
        return;
    selectedTag.value = keyword;
    await fetchReview(keyword, 'review_pick_by_tag');
}
async function switchReview() {
    if (!payload.value || switching.value)
        return;
    await fetchReview(selectedTag.value, 'review_switch');
}
async function fetchReview(tag, action) {
    if (!payload.value)
        return;
    switching.value = true;
    try {
        const { data } = await publicApi.switchReview(String(route.params.token), {
            tag: tag || undefined,
            sessionId: payload.value.sessionId
        });
        payload.value.review = data.data.review;
        payload.value.remainingDispatchableCount = data.data.remainingDispatchableCount;
        await publicApi.createEvent(String(route.params.token), {
            sessionId: payload.value.sessionId,
            reviewItemId: payload.value.review.id,
            actionType: action
            // 不把关键词标签塞进 platformCode（那是给平台跳转事件用的），避免污染分析
        });
    }
    catch (err) {
        alert(err?.response?.data?.message || '暂无推荐文案，请稍后再试');
    }
    finally {
        switching.value = false;
    }
}
async function copyReview() {
    const text = editedContent.value.trim();
    if (!payload.value || !text)
        return;
    try {
        await navigator.clipboard.writeText(text);
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
async function jump(link) {
    const text = editedContent.value.trim();
    if (!payload.value || !text)
        return;
    try {
        await navigator.clipboard.writeText(text);
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
    if (__VLS_ctx.payload.keywords && __VLS_ctx.payload.keywords.length) {
        __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            ...{ style: {} },
        });
        __VLS_asFunctionalElement(__VLS_intrinsicElements.p, __VLS_intrinsicElements.p)({
            ...{ class: "muted" },
            ...{ style: {} },
        });
        __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
            ...{ class: "row" },
            ...{ style: {} },
        });
        for (const [kw] of __VLS_getVForSourceType((__VLS_ctx.payload.keywords))) {
            __VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
                ...{ onClick: (...[$event]) => {
                        if (!!(__VLS_ctx.loading))
                            return;
                        if (!!(__VLS_ctx.error))
                            return;
                        if (!(__VLS_ctx.payload))
                            return;
                        if (!(__VLS_ctx.payload.keywords && __VLS_ctx.payload.keywords.length))
                            return;
                        __VLS_ctx.pickByTag(kw.keyword);
                    } },
                key: (kw.id),
                ...{ class: ({ secondary: __VLS_ctx.selectedTag !== kw.keyword }) },
                disabled: (__VLS_ctx.switching),
            });
            (kw.keyword);
        }
    }
    __VLS_asFunctionalElement(__VLS_intrinsicElements.p, __VLS_intrinsicElements.p)({
        ...{ class: "muted" },
        ...{ style: {} },
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.textarea, __VLS_intrinsicElements.textarea)({
        value: (__VLS_ctx.editedContent),
        rows: "8",
        ...{ style: {} },
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
        ...{ class: "row" },
        ...{ style: {} },
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
        ...{ onClick: (__VLS_ctx.copyReview) },
        disabled: (!__VLS_ctx.editedContent.trim()),
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
        ...{ onClick: (__VLS_ctx.switchReview) },
        ...{ class: "secondary" },
        disabled: (__VLS_ctx.switching),
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
                disabled: (!__VLS_ctx.editedContent.trim()),
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
/** @type {__VLS_StyleScopedClasses['muted']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
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
            switching: switching,
            error: error,
            payload: payload,
            selectedTag: selectedTag,
            editedContent: editedContent,
            pickByTag: pickByTag,
            switchReview: switchReview,
            copyReview: copyReview,
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
