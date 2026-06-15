import { onMounted, reactive, ref } from 'vue';
import { merchantApi } from '../../api/merchant';
import { useAuthStore } from '../../stores/auth';
const auth = useAuthStore();
const storeForm = reactive({
    storeName: '',
    industryType: '',
    storeIntro: '',
    address: '',
    primaryPlatformStyle: 'xiaohongshu',
    brandTone: ''
});
const keyword = ref('');
const imageUrl = ref('');
const platformForm = reactive({ platformCode: 'meituan', platformName: '美团', buttonText: '去美团评论', targetUrl: 'https://www.meituan.com/', backupUrl: '', sortNo: 1, status: 1 });
const reviewText = ref('');
const keywords = ref([]);
const images = ref([]);
const links = ref([]);
const reviews = ref([]);
const tasks = ref([]);
async function loadAll() {
    const [storeRes, keywordRes, imageRes, linkRes, reviewRes, taskRes] = await Promise.all([
        merchantApi.getStoreDetail(),
        merchantApi.listKeywords(),
        merchantApi.listImages(),
        merchantApi.listPlatformLinks(),
        merchantApi.listReviews(),
        merchantApi.listTasks()
    ]);
    Object.assign(storeForm, storeRes.data.data);
    keywords.value = keywordRes.data.data;
    images.value = imageRes.data.data;
    links.value = linkRes.data.data;
    reviews.value = reviewRes.data.data;
    tasks.value = taskRes.data.data;
}
async function saveStore() {
    await merchantApi.updateStoreDetail(storeForm);
    alert('门店信息已保存');
}
async function addKeyword() {
    if (!keyword.value)
        return;
    await merchantApi.createKeyword({ keyword: keyword.value, sortNo: keywords.value.length + 1 });
    keyword.value = '';
    await loadAll();
}
async function addImage() {
    if (!imageUrl.value)
        return;
    await merchantApi.createImage({ imageUrl: imageUrl.value, thumbnailUrl: imageUrl.value, sortNo: images.value.length + 1 });
    imageUrl.value = '';
    await loadAll();
}
async function addPlatformLink() {
    await merchantApi.createPlatformLink(platformForm);
    await loadAll();
}
async function addReview() {
    if (!reviewText.value)
        return;
    await merchantApi.createReview({ content: reviewText.value, status: 'available' });
    reviewText.value = '';
    await loadAll();
}
async function generateReviews() {
    await merchantApi.generateReviews(10);
    await loadAll();
}
function logout() {
    auth.clear();
    location.href = '/merchant/login';
}
onMounted(loadAll);
debugger; /* PartiallyEnd: #3632/scriptSetup.vue */
const __VLS_ctx = {};
let __VLS_components;
let __VLS_directives;
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "page" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "row" },
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.h1, __VLS_intrinsicElements.h1)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
    ...{ onClick: (__VLS_ctx.logout) },
    ...{ class: "secondary" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "grid-2" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "card" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "门店名称",
});
(__VLS_ctx.storeForm.storeName);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "行业类型",
});
(__VLS_ctx.storeForm.industryType);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "门店地址",
});
(__VLS_ctx.storeForm.address);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "主平台风格",
});
(__VLS_ctx.storeForm.primaryPlatformStyle);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.textarea, __VLS_intrinsicElements.textarea)({
    value: (__VLS_ctx.storeForm.storeIntro),
    placeholder: "门店简介",
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "品牌调性",
});
(__VLS_ctx.storeForm.brandTone);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
    ...{ onClick: (__VLS_ctx.saveStore) },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "card" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
    ...{ onClick: (__VLS_ctx.generateReviews) },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.table, __VLS_intrinsicElements.table)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.thead, __VLS_intrinsicElements.thead)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.tr, __VLS_intrinsicElements.tr)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.th, __VLS_intrinsicElements.th)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.th, __VLS_intrinsicElements.th)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.th, __VLS_intrinsicElements.th)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.th, __VLS_intrinsicElements.th)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.tbody, __VLS_intrinsicElements.tbody)({});
for (const [task] of __VLS_getVForSourceType((__VLS_ctx.tasks))) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.tr, __VLS_intrinsicElements.tr)({
        key: (task.id),
    });
    __VLS_asFunctionalElement(__VLS_intrinsicElements.td, __VLS_intrinsicElements.td)({});
    (task.id);
    __VLS_asFunctionalElement(__VLS_intrinsicElements.td, __VLS_intrinsicElements.td)({});
    (task.triggerType);
    __VLS_asFunctionalElement(__VLS_intrinsicElements.td, __VLS_intrinsicElements.td)({});
    (task.status);
    __VLS_asFunctionalElement(__VLS_intrinsicElements.td, __VLS_intrinsicElements.td)({});
    (task.successCount);
}
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "grid-2" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "card" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "row" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "新增关键词",
});
(__VLS_ctx.keyword);
__VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
    ...{ onClick: (__VLS_ctx.addKeyword) },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.ul, __VLS_intrinsicElements.ul)({});
for (const [item] of __VLS_getVForSourceType((__VLS_ctx.keywords))) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.li, __VLS_intrinsicElements.li)({
        key: (item.id),
    });
    (item.keyword);
}
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "card" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "row" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "图片 URL",
});
(__VLS_ctx.imageUrl);
__VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
    ...{ onClick: (__VLS_ctx.addImage) },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "row" },
});
for (const [item] of __VLS_getVForSourceType((__VLS_ctx.images))) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.img)({
        key: (item.id),
        src: (item.thumbnailUrl || item.imageUrl),
        ...{ style: {} },
    });
}
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "grid-2" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "card" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "平台编码",
});
(__VLS_ctx.platformForm.platformCode);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "平台名称",
});
(__VLS_ctx.platformForm.platformName);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "按钮文案",
});
(__VLS_ctx.platformForm.buttonText);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "主跳转链接",
});
(__VLS_ctx.platformForm.targetUrl);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.input)({
    placeholder: "备用链接",
});
(__VLS_ctx.platformForm.backupUrl);
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
    ...{ onClick: (__VLS_ctx.addPlatformLink) },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.ul, __VLS_intrinsicElements.ul)({});
for (const [item] of __VLS_getVForSourceType((__VLS_ctx.links))) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.li, __VLS_intrinsicElements.li)({
        key: (item.id),
    });
    (item.buttonText);
    (item.targetUrl);
}
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ class: "card" },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.h2, __VLS_intrinsicElements.h2)({});
__VLS_asFunctionalElement(__VLS_intrinsicElements.textarea, __VLS_intrinsicElements.textarea)({
    value: (__VLS_ctx.reviewText),
    placeholder: "新增手工评价",
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.div, __VLS_intrinsicElements.div)({
    ...{ style: {} },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.button, __VLS_intrinsicElements.button)({
    ...{ onClick: (__VLS_ctx.addReview) },
});
__VLS_asFunctionalElement(__VLS_intrinsicElements.ul, __VLS_intrinsicElements.ul)({});
for (const [item] of __VLS_getVForSourceType((__VLS_ctx.reviews.slice(0, 8)))) {
    __VLS_asFunctionalElement(__VLS_intrinsicElements.li, __VLS_intrinsicElements.li)({
        key: (item.id),
    });
    (item.content);
}
/** @type {__VLS_StyleScopedClasses['page']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
/** @type {__VLS_StyleScopedClasses['secondary']} */ ;
/** @type {__VLS_StyleScopedClasses['grid-2']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['grid-2']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
/** @type {__VLS_StyleScopedClasses['row']} */ ;
/** @type {__VLS_StyleScopedClasses['grid-2']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
/** @type {__VLS_StyleScopedClasses['card']} */ ;
var __VLS_dollars;
const __VLS_self = (await import('vue')).defineComponent({
    setup() {
        return {
            storeForm: storeForm,
            keyword: keyword,
            imageUrl: imageUrl,
            platformForm: platformForm,
            reviewText: reviewText,
            keywords: keywords,
            images: images,
            links: links,
            reviews: reviews,
            tasks: tasks,
            saveStore: saveStore,
            addKeyword: addKeyword,
            addImage: addImage,
            addPlatformLink: addPlatformLink,
            addReview: addReview,
            generateReviews: generateReviews,
            logout: logout,
        };
    },
});
export default (await import('vue')).defineComponent({
    setup() {
        return {};
    },
});
; /* PartiallyEnd: #4569/main.vue */
