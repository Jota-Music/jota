export function secondsToTime(time: number) {
    return String(Math.floor(time / 60)).padStart(2, "0") + ":" + String(Math.floor(time % 60)).padStart(2, "0");
}