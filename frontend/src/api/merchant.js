import http from './http';
export const merchantApi = {
    login(payload) {
        return http.post('/merchant/auth/login', payload);
    },
    getStoreDetail() {
        return http.get('/merchant/store/detail');
    },
    updateStoreDetail(payload) {
        return http.put('/merchant/store/detail', payload);
    },
    listKeywords() {
        return http.get('/merchant/store/keywords');
    },
    createKeyword(payload) {
        return http.post('/merchant/store/keywords', payload);
    },
    deleteKeyword(id) {
        return http.delete(`/merchant/store/keywords/${id}`);
    },
    listImages() {
        return http.get('/merchant/store/images');
    },
    createImage(payload) {
        return http.post('/merchant/store/images/upload', payload);
    },
    deleteImage(id) {
        return http.delete(`/merchant/store/images/${id}`);
    },
    listPlatformLinks() {
        return http.get('/merchant/store/platform-links');
    },
    createPlatformLink(payload) {
        return http.post('/merchant/store/platform-links', payload);
    },
    updatePlatformLinkStatus(id, status) {
        return http.put(`/merchant/store/platform-links/${id}/status`, { status });
    },
    deletePlatformLink(id) {
        return http.delete(`/merchant/store/platform-links/${id}`);
    },
    listReviews() {
        return http.get('/merchant/reviews');
    },
    createReview(payload) {
        return http.post('/merchant/reviews', payload);
    },
    deleteReview(id) {
        return http.delete(`/merchant/reviews/${id}`);
    },
    generateReviews(targetCount = 10) {
        return http.post('/merchant/reviews/generate', { targetCount });
    },
    listTasks() {
        return http.get('/merchant/review-generation-tasks');
    }
};
