import { SegmentedIndex} from "k6/x/segment";

export default function() {
    var index = new SegmentedIndex();
    var v =index.next()
    console.log(v.scaled, v.unscaled);
    v =index.next()
    console.log(v.scaled, v.unscaled);
    v =index.next()
    console.log(v.scaled, v.unscaled);
    v =index.next()
    console.log(v.scaled, v.unscaled);
    v =index.next()
    console.log(v.scaled, v.unscaled);
    v =index.next()
    console.log(v.scaled, v.unscaled);
}
