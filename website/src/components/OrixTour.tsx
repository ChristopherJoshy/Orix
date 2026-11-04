import { useState } from 'react';
import { Modal } from './Modal';
import { Button } from './ui';
import { PixelStatus, type PixelState } from './PixelStatus';
const steps: { title: string; text: string; scene: PixelState }[] = [
  { title: 'Set your route', text: 'Choose your start and destination from the place suggestions. Orix looks for compatible travellers starting within 300 metres and heading within 100 m of each other.', scene: 'searching' },
  { title: 'Meet your group', text: 'Check the people, pickup plan and time when a pool is ready. Use your group conversation to coordinate before leaving.', scene: 'found' },
  { title: 'Choose what works for you', text: 'Set safety preferences and whether lifts are suitable. To offer a lift, register your route, time and seats. Block contacts by email or open a private report conversation when you need help.', scene: 'lift-offered' },
];
export function OrixTour({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [step, setStep] = useState(0);
  return <Modal open={open} onOpenChange={(value) => { if (!value) onClose(); }} title={steps[step].title}>
    <PixelStatus state={steps[step].scene} label={steps[step].title} />
    <p>{steps[step].text}</p>
    <p className="muted">{step + 1} of {steps.length}</p>
    <div className="intro-actions">
      {step > 0 && <Button type="button" variant="ghost" onClick={() => setStep(step - 1)}>Back</Button>}
      <Button type="button" onClick={() => step < steps.length - 1 ? setStep(step + 1) : onClose()}>{step < steps.length - 1 ? 'Next' : 'Got it'}</Button>
    </div>
  </Modal>;
}

export function TourLauncher() {
  const [open, setOpen] = useState(false);
  return <><Button type="button" variant="secondary" onClick={() => setOpen(true)}>Replay the tour</Button><OrixTour open={open} onClose={() => setOpen(false)} /></>;
}
