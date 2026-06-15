import http from './http';
export const publicApi = {
    initLanding(token) {
        return http.get(`/public/landing/${token}/init`);
    },
    switchReview(token, payload) {
        return http.post(`/public/landing/${token}/switch-review`, payload);
    },
    createEvent(token, payload) {
        return http.post(`/public/landing/${token}/events`, payload);
    }
};
