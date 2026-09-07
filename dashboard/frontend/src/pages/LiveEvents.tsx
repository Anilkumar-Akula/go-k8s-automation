import EventList from "../components/EventList";

export default function LiveEvents() {
  return (
    <section>
      <h1>Live Events</h1>
      <EventList limit={50} />
    </section>
  );
}
