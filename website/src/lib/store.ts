import { create } from "zustand";
import { persist } from "zustand/middleware";
interface InterfacePreferences {
  motion: boolean;
  setMotion: (value: boolean) => void;
}
/** Interface preferences only. Account, journey and moderation state live on the API. */
export const useStore = create<InterfacePreferences>()(
  persist((set) => ({ motion: true, setMotion: (motion) => set({ motion }) }), {
    name: "orix-web-ui-v1",
  }),
);
